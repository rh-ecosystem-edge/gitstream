// Code generated by MockGen. DO NOT EDIT.
// Source: getter.go

// Package intents is a generated GoMock package.
package intents

import (
	context "context"
	reflect "reflect"
	time "time"

	v5 "github.com/go-git/go-git/v5"
	plumbing "github.com/go-git/go-git/v5/plumbing"
	gomock "github.com/golang/mock/gomock"
	github "github.com/qbarrand/gitstream/internal/github"
)

// MockGetter is a mock of Getter interface.
type MockGetter struct {
	ctrl     *gomock.Controller
	recorder *MockGetterMockRecorder
}

// MockGetterMockRecorder is the mock recorder for MockGetter.
type MockGetterMockRecorder struct {
	mock *MockGetter
}

// NewMockGetter creates a new mock instance.
func NewMockGetter(ctrl *gomock.Controller) *MockGetter {
	mock := &MockGetter{ctrl: ctrl}
	mock.recorder = &MockGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGetter) EXPECT() *MockGetterMockRecorder {
	return m.recorder
}

// FromGitHubIssues mocks base method.
func (m *MockGetter) FromGitHubIssues(ctx context.Context, rn *github.RepoName) (CommitIntents, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FromGitHubIssues", ctx, rn)
	ret0, _ := ret[0].(CommitIntents)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FromGitHubIssues indicates an expected call of FromGitHubIssues.
func (mr *MockGetterMockRecorder) FromGitHubIssues(ctx, rn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FromGitHubIssues", reflect.TypeOf((*MockGetter)(nil).FromGitHubIssues), ctx, rn)
}

// FromGitHubOpenPRs mocks base method.
func (m *MockGetter) FromGitHubOpenPRs(ctx context.Context, rn *github.RepoName) (CommitIntents, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FromGitHubOpenPRs", ctx, rn)
	ret0, _ := ret[0].(CommitIntents)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FromGitHubOpenPRs indicates an expected call of FromGitHubOpenPRs.
func (mr *MockGetterMockRecorder) FromGitHubOpenPRs(ctx, rn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FromGitHubOpenPRs", reflect.TypeOf((*MockGetter)(nil).FromGitHubOpenPRs), ctx, rn)
}

// FromLocalGitRepo mocks base method.
func (m *MockGetter) FromLocalGitRepo(ctx context.Context, repo *v5.Repository, from plumbing.Hash, since *time.Time) (CommitIntents, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FromLocalGitRepo", ctx, repo, from, since)
	ret0, _ := ret[0].(CommitIntents)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FromLocalGitRepo indicates an expected call of FromLocalGitRepo.
func (mr *MockGetterMockRecorder) FromLocalGitRepo(ctx, repo, from, since interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FromLocalGitRepo", reflect.TypeOf((*MockGetter)(nil).FromLocalGitRepo), ctx, repo, from, since)
}