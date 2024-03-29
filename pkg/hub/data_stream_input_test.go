package hub

import (
	"bytes"
	"errors"
	"io"
	"testing"

	. "github.com/franela/goblin"
	"github.com/golang/mock/gomock"
	mock_datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage/mocks"
	mock_globals "github.com/thebartekbanach/imcaxy/test/mocks"
)

func TestDataStreamInput(t *testing.T) {
	g := Goblin(t)

	g.Describe("DataStreamInput", func() {
		g.It("Should correctly write data to underlying storage", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			dataToWrite := []byte{0x1, 0x2, 0x3}
			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Write("test", dataToWrite).Return(3, nil).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			n, err := stream.Write(dataToWrite)

			g.Assert(n).Equal(3)
			g.Assert(err).Equal(nil)
		})

		g.It("Should forward write error from Write method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			dataToWrite := []byte{0x1, 0x2, 0x3}
			testError := errors.New("test error")
			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Write("test", dataToWrite).Return(0, testError)

			stream := newDataStreamInput("test", mockWriter)
			_, err := stream.Write(dataToWrite)

			g.Assert(err).Equal(testError)
		})

		g.It("Should correctly close resource", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", nil).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			stream.Close(nil)
		})

		g.It("Should close resource and forward given error", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			testError := errors.New("test error")
			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", testError).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			stream.Close(testError)
		})

		g.It("Should return error returned by storage while closing resource", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			testError := errors.New("test error")
			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", nil).Return(testError)

			stream := newDataStreamInput("test", mockWriter)
			err := stream.Close(nil)

			g.Assert(err).Equal(testError)
		})

		g.It("Should correctly read all data from reader using ReadFrom method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Write("test", []byte{0x1, 0x2, 0x3}).Return(3, nil).Times(1)
			mockWriter.EXPECT().Write("test", []byte{0x4, 0x5, 0x6}).Return(3, nil).Times(1)

			mockReader := mock_globals.NewMockReader(mockCtrl)
			mockReader.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
				n := copy(p, []byte{0x1, 0x2, 0x3})
				return n, nil
			}).Times(1)
			mockReader.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
				n := copy(p, []byte{0x4, 0x5, 0x6})
				return n, io.EOF
			}).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			stream.ReadFrom(mockReader)
		})

		g.It("Should return and forward reading error while using ReadFrom method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Write("test", []byte{0x1, 0x2, 0x3}).Return(3, nil).Times(1)

			mockReader := mock_globals.NewMockReader(mockCtrl)
			mockReader.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
				n := copy(p, []byte{0x1, 0x2, 0x3})
				return n, nil
			}).Times(1)
			mockReader.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
				n := copy(p, []byte{0x4, 0x5, 0x6})
				return n, io.ErrUnexpectedEOF
			}).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			_, err := stream.ReadFrom(mockReader)

			g.Assert(err).Equal(io.ErrUnexpectedEOF)
		})

		g.It("Should return writing error while using ReadFrom method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Write("test", []byte{0x1, 0x2, 0x3}).Return(3, io.ErrUnexpectedEOF).Times(1)

			mockReader := mock_globals.NewMockReader(mockCtrl)
			mockReader.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
				n := copy(p, []byte{0x1, 0x2, 0x3})
				return n, nil
			}).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			_, err := stream.ReadFrom(mockReader)

			g.Assert(err).Equal(io.ErrUnexpectedEOF)
		})

		g.It("Should not forward io.EOF error when using Close method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", nil).Return(nil)

			stream := newDataStreamInput("test", mockWriter)
			stream.Close(io.EOF)
		})

		g.It("Should return ErrStreamAlreadyClosed if trying to close stream that is already closed", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", nil).Return(nil).Times(1)

			stream := newDataStreamInput("test", mockWriter)
			stream.Close(nil)

			err := stream.Close(nil)

			g.Assert(err).Equal(ErrStreamAlreadyClosed)
		})

		g.It("Should return ErrStreamClosedForWriting when trying to write data to closed stream", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", nil).Return(nil)

			stream := newDataStreamInput("test", mockWriter)
			stream.Close(nil)

			_, err := stream.Write([]byte{0x1, 0x2, 0x3})

			g.Assert(err).Equal(ErrStreamClosedForWriting)
		})

		g.It("Should return ErrStreamClosedForWriting when trying to use ReadFrom method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockWriter := mock_datahubstorage.NewMockWriter(mockCtrl)
			mockWriter.EXPECT().Close("test", nil).Return(nil)

			stream := newDataStreamInput("test", mockWriter)
			stream.Close(nil)

			reader := bytes.NewReader([]byte{0x1, 0x2, 0x3})
			_, err := stream.ReadFrom(reader)

			g.Assert(err).Equal(ErrStreamClosedForWriting)
		})
	})
}
