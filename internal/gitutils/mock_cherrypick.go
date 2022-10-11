// Code generated by MockGen. DO NOT EDIT.
// Source: cherrypick.go

// Package gitutils is a generated GoMock package.
package gitutils

import (
	context "context"
	reflect "reflect"

	git "github.com/go-git/go-git/v5"
	object "github.com/go-git/go-git/v5/plumbing/object"
	logr "github.com/go-logr/logr"
	gomock "github.com/golang/mock/gomock"
)

// MockCherryPicker is a mock of CherryPicker interface.
type MockCherryPicker struct {
	ctrl     *gomock.Controller
	recorder *MockCherryPickerMockRecorder
}

// MockCherryPickerMockRecorder is the mock recorder for MockCherryPicker.
type MockCherryPickerMockRecorder struct {
	mock *MockCherryPicker
}

// NewMockCherryPicker creates a new mock instance.
func NewMockCherryPicker(ctrl *gomock.Controller) *MockCherryPicker {
	mock := &MockCherryPicker{ctrl: ctrl}
	mock.recorder = &MockCherryPickerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCherryPicker) EXPECT() *MockCherryPickerMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockCherryPicker) Run(ctx context.Context, repo *git.Repository, repoPath string, commit *object.Commit) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx, repo, repoPath, commit)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockCherryPickerMockRecorder) Run(ctx, repo, repoPath, commit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockCherryPicker)(nil).Run), ctx, repo, repoPath, commit)
}

// MockExecutor is a mock of Executor interface.
type MockExecutor struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorMockRecorder
}

// MockExecutorMockRecorder is the mock recorder for MockExecutor.
type MockExecutorMockRecorder struct {
	mock *MockExecutor
}

// NewMockExecutor creates a new mock instance.
func NewMockExecutor(ctrl *gomock.Controller) *MockExecutor {
	mock := &MockExecutor{ctrl: ctrl}
	mock.recorder = &MockExecutorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutor) EXPECT() *MockExecutorMockRecorder {
	return m.recorder
}

// RunCommand mocks base method.
func (m *MockExecutor) RunCommand(ctx context.Context, logger logr.Logger, bin, dir string, args ...string) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, logger, bin, dir}
	for _, a := range args {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "RunCommand", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// RunCommand indicates an expected call of RunCommand.
func (mr *MockExecutorMockRecorder) RunCommand(ctx, logger, bin, dir interface{}, args ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, logger, bin, dir}, args...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunCommand", reflect.TypeOf((*MockExecutor)(nil).RunCommand), varargs...)
}
