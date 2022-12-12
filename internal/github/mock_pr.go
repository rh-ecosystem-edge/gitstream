// Code generated by MockGen. DO NOT EDIT.
// Source: pr.go

// Package github is a generated GoMock package.
package github

import (
	context "context"
	reflect "reflect"

	object "github.com/go-git/go-git/v5/plumbing/object"
	gomock "github.com/golang/mock/gomock"
	github "github.com/google/go-github/v47/github"
)

// MockPRHelper is a mock of PRHelper interface.
type MockPRHelper struct {
	ctrl     *gomock.Controller
	recorder *MockPRHelperMockRecorder
}

// MockPRHelperMockRecorder is the mock recorder for MockPRHelper.
type MockPRHelperMockRecorder struct {
	mock *MockPRHelper
}

// NewMockPRHelper creates a new mock instance.
func NewMockPRHelper(ctrl *gomock.Controller) *MockPRHelper {
	mock := &MockPRHelper{ctrl: ctrl}
	mock.recorder = &MockPRHelperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPRHelper) EXPECT() *MockPRHelperMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockPRHelper) Create(ctx context.Context, branch, base, upstreamURL string, commit *object.Commit, draft bool) (*github.PullRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, branch, base, upstreamURL, commit, draft)
	ret0, _ := ret[0].(*github.PullRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockPRHelperMockRecorder) Create(ctx, branch, base, upstreamURL, commit, draft interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockPRHelper)(nil).Create), ctx, branch, base, upstreamURL, commit, draft)
}

// ListAllOpen mocks base method.
func (m *MockPRHelper) ListAllOpen(ctx context.Context, filter PRFilterFunc) ([]*github.PullRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAllOpen", ctx, filter)
	ret0, _ := ret[0].([]*github.PullRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAllOpen indicates an expected call of ListAllOpen.
func (mr *MockPRHelperMockRecorder) ListAllOpen(ctx, filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAllOpen", reflect.TypeOf((*MockPRHelper)(nil).ListAllOpen), ctx, filter)
}

// MakeReady mocks base method.
func (m *MockPRHelper) MakeReady(ctx context.Context, pr *github.PullRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeReady", ctx, pr)
	ret0, _ := ret[0].(error)
	return ret0
}

// MakeReady indicates an expected call of MakeReady.
func (mr *MockPRHelperMockRecorder) MakeReady(ctx, pr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeReady", reflect.TypeOf((*MockPRHelper)(nil).MakeReady), ctx, pr)
}