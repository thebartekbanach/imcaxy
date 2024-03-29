package imaginaryprocessor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/franela/goblin"
	"github.com/golang/mock/gomock"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
	"github.com/thebartekbanach/imcaxy/pkg/processor"
	testutils "github.com/thebartekbanach/imcaxy/test/utils"
)

type httpResponseBody struct {
	io.Reader
	readError error
}

func (body *httpResponseBody) Read(p []byte) (n int, err error) {
	if body.readError != nil {
		return 0, body.readError
	}

	return body.Reader.Read(p)
}

func (body *httpResponseBody) Close() error {
	return nil
}

func testReqFunc(
	statusCode int,
	response []byte,
	callError, responseBodyError error,
	includeContentType bool,
	responseSize func(response []byte) string,
	requestAssert func(req *http.Request),
) httpRequestFunc {
	return func(req *http.Request) (*http.Response, error) {
		requestAssert(req)

		if callError != nil {
			return nil, callError
		}

		reader := bytes.NewReader(response)
		body := httpResponseBody{reader, responseBodyError}

		headers := http.Header{}

		responseSizeHeader := responseSize(response)
		if responseSizeHeader != "" {
			headers.Add("Content-Length", responseSizeHeader)
		}

		if includeContentType {
			headers.Add("Content-Type", "image/png")
		}

		return &http.Response{
			StatusCode: statusCode,
			Body:       &body,
			Header:     headers,
		}, nil
	}
}

func noAssertions(req *http.Request) {}

func normalResponseSize(response []byte) string {
	return strconv.Itoa(len(response))
}

func stringResponseSize(responseString string) func(response []byte) string {
	return func(_ []byte) string {
		return responseString
	}
}

func TestImaginaryProcessor(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Processor", func() {
		g.Describe("ParseRequest", func() {
			g.It("Should correctly destruct given request path into request information", func() {
				config := Config{}

				processor := NewProcessor(config)
				result, _ := processor.ParseRequest("/crop?abc=1&def=2&url=http://google.com/image.jpg")

				g.Assert(result.ProcessorEndpoint).Equal("/crop")
				g.Assert(result.SourceImageURL).Equal("http://google.com/image.jpg")
				g.Assert(result.ProcessingParams).Equal(map[string][]string{
					"abc": {"1"},
					"def": {"2"},
					"url": {"http://google.com/image.jpg"},
				})
			})

			g.It("Should generate correct unique checksum of request", func() {
				config := Config{}

				processor := NewProcessor(config)
				firstResult, _ := processor.ParseRequest("/crop?abc=1&def=2&url=http://google.com/image.jpg")
				secondResult, _ := processor.ParseRequest("/crop?abc=1&url=http://google.com/image.jpg&def=2")

				g.Assert(firstResult.Signature).Equal(secondResult.Signature)
			})

			g.It("Should return error if sourceImageURL not found in request", func() {
				config := Config{}

				processor := NewProcessor(config)
				_, err := processor.ParseRequest("/crop?abc=1&def=2")

				g.Assert(err).IsNotNil()
			})

			g.It("Should return error if processorEndpoint is not correct", func() {
				config := Config{}

				processor := NewProcessor(config)
				_, err := processor.ParseRequest("/unknown?abc=1&def=2&url=http://google.com/image.jpg")

				g.Assert(err).IsNotNil()
			})
		})

		g.Describe("ProcessImage", func() {
			g.It("Should correctly construct and send request to imaginary service", func() {
				mockCtrl := gomock.NewController(g)
				defer mockCtrl.Finish()

				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, true, normalResponseSize, func(req *http.Request) {
					g.Assert(req.Method).Equal(http.MethodGet)
					g.Assert(req.URL.Host).Equal("http://localhost:3000")
					g.Assert(req.URL.Path).Equal("/crop")
				})

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				contentType, _, _ := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(contentType).Equal("image/png")
			})

			g.It("Should write all contents of imaginary service response into data stream input", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, true, normalResponseSize, noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				proc.ProcessImage(ctx, parsedRequest, &inputStream)

				inputStream.Wait()
				g.Assert(inputStream.SafelyGetDataSegment(0)).Equal(testData)
			})

			g.It("Should return error when http request returns error", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, nil, io.ErrUnexpectedEOF, nil, true, normalResponseSize, noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				_, _, err := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(err).Equal(io.ErrUnexpectedEOF)
			})

			g.It("Should return error if imaginary service responds with not-200 error code", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(500, testData, nil, nil, true, normalResponseSize, noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				_, _, err := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(err).Equal(ErrResponseStatusNotOK)
			})

			g.It("Should return error if imaginary service response does not include Content-Type header", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, false, normalResponseSize, noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				_, _, err := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(err).Equal(ErrUnknownContentType)
			})

			g.It("Should return error if imaginary service response does not include Content-Length header", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, true, stringResponseSize(""), noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				_, _, err := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(err).Equal(ErrUnknownContentLength)
			})

			g.It("Should return error if responses Content-Length header is zero", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, true, stringResponseSize("0"), noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				_, _, err := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(err).Equal(ErrUnknownContentLength)
			})

			g.It("Should return error if responses Content-Length header is not correct number", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, true, stringResponseSize("incorrect"), noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				_, _, err := proc.ProcessImage(ctx, parsedRequest, &inputStream)

				g.Assert(err).Equal(ErrUnknownContentLength)
			})

			g.It("Should close input data stream with error that ocurred while fetching given image", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, nil, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, io.ErrUnexpectedEOF, true, normalResponseSize, noAssertions)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				proc := Processor{config, requestMaker}
				proc.ProcessImage(ctx, parsedRequest, &inputStream)

				inputStream.Wait()
				g.Assert(inputStream.ForwardedError).Equal(io.ErrUnexpectedEOF)
			})
		})
	})
}

func loadTestFile(t *testing.T, w io.Writer) {
	file, err := os.Open("./../../../test/data/image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	io.Copy(w, file)
}

func getResultFileContents(t *testing.T) []byte {
	file, err := os.Open("./../../../test/data/processed.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	return contents
}

func TestImaginaryProcessorIntegration_ShouldCorrectlyProcessImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ImaginaryProcessor integration test")
	}

	testServer := testutils.NewTestHttpServer()
	testServer.HandleFunc("/image.jpg", func(w http.ResponseWriter, r *http.Request) {
		loadTestFile(t, w)
		w.Header().Set("Content-Type", "image/jpeg")
		r.Body.Close()
	})

	port := testServer.Start(t)
	resourceURL := fmt.Sprintf("http://IntegrationTests.Imcaxy.Server:%d/image.jpg", port)

	processor := NewProcessor(Config{ImaginaryServiceURL: "IntegrationTests.Imcaxy.Imaginary:8080"})
	req, err := processor.ParseRequest("/crop?width=300&height=300&url=" + resourceURL)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := datahubstorage.NewStorage()
	datahub := hub.NewDataHub(storage)
	datahub.StartMonitors(ctx)

	outputStream, inputStream, err := datahub.GetOrCreateStream("test")
	if err != nil {
		t.Fatal(err)
	}

	contentType, size, err := processor.ProcessImage(ctx, req, inputStream)
	if err != nil {
		t.Fatal(err)
	}

	if contentType != "image/jpeg" {
		t.Fatalf("expected content type to be image/jpeg, got %s", contentType)
	}

	data, err := ioutil.ReadAll(outputStream)
	if err != nil {
		t.Fatal(err)
	}

	if int64(len(data)) != size {
		t.Fatalf("expected size to be %d, got %d", size, len(data))
	}

	expectedResult := getResultFileContents(t)
	if !bytes.Equal(data, expectedResult) {
		t.Fatalf("data got from ImaginaryProcessor is not equal to expected result |||||||")
	}
}

func TestImaginaryProcessorIntegration_ShouldReturnErrorWhenResponseCodeIsNotCorrect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ImaginaryProcessor integration test")
	}

	testServer := testutils.NewTestHttpServer()
	testServer.HandleFunc("/image.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	port := testServer.Start(t)
	resourceURL := fmt.Sprintf("http://IntegrationTests.Imcaxy.Server:%d/image.jpg", port)

	processor := NewProcessor(Config{ImaginaryServiceURL: "IntegrationTests.Imcaxy.Imaginary:8080"})
	req, err := processor.ParseRequest("/crop?width=300&height=300&url=" + resourceURL)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := datahubstorage.NewStorage()
	datahub := hub.NewDataHub(storage)
	datahub.StartMonitors(ctx)

	_, inputStream, err := datahub.GetOrCreateStream("test")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = processor.ProcessImage(ctx, req, inputStream)
	if err != ErrResponseStatusNotOK {
		t.Fatalf("expected error to be ErrResponseStatusNotOK, got %v", err)
	}
}
