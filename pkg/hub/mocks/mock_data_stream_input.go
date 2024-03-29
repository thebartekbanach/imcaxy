// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/thebartekbanach/imcaxy/pkg/hub (interfaces: DataStreamInput)

// Package mock_hub is a generated GoMock package.
package mock_hub

import (
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockDataStreamInput is a mock of DataStreamInput interface.
type MockDataStreamInput struct {
	ctrl     *gomock.Controller
	recorder *MockDataStreamInputMockRecorder
}

// MockDataStreamInputMockRecorder is the mock recorder for MockDataStreamInput.
type MockDataStreamInputMockRecorder struct {
	mock *MockDataStreamInput
}

// NewMockDataStreamInput creates a new mock instance.
func NewMockDataStreamInput(ctrl *gomock.Controller) *MockDataStreamInput {
	mock := &MockDataStreamInput{ctrl: ctrl}
	mock.recorder = &MockDataStreamInputMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDataStreamInput) EXPECT() *MockDataStreamInputMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockDataStreamInput) Close(arg0 error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockDataStreamInputMockRecorder) Close(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockDataStreamInput)(nil).Close), arg0)
}

// ReadFrom mocks base method.
func (m *MockDataStreamInput) ReadFrom(arg0 io.Reader) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFrom", arg0)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFrom indicates an expected call of ReadFrom.
func (mr *MockDataStreamInputMockRecorder) ReadFrom(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFrom", reflect.TypeOf((*MockDataStreamInput)(nil).ReadFrom), arg0)
}

// Write mocks base method.
func (m *MockDataStreamInput) Write(arg0 []byte) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Write indicates an expected call of Write.
func (mr *MockDataStreamInputMockRecorder) Write(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockDataStreamInput)(nil).Write), arg0)
}
