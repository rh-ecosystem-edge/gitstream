// Code generated by MockGen. DO NOT EDIT.
// Source: differ.go

// Package gitutils is a generated GoMock package.
package gitutils

import (
	context "context"
	reflect "reflect"
	time "time"

	git "github.com/go-git/go-git/v5"
	object "github.com/go-git/go-git/v5/plumbing/object"
	gomock "github.com/golang/mock/gomock"
	config "github.com/qbarrand/gitstream/internal/config"
	github "github.com/qbarrand/gitstream/internal/github"
)

// MockDiffer is a mock of Differ interface.
type MockDiffer struct {
	ctrl     *gomock.Controller
	recorder *MockDifferMockRecorder
}

// MockDifferMockRecorder is the mock recorder for MockDiffer.
type MockDifferMockRecorder struct {
	mock *MockDiffer
}

// NewMockDiffer creates a new mock instance.
func NewMockDiffer(ctrl *gomock.Controller) *MockDiffer {
	mock := &MockDiffer{ctrl: ctrl}
	mock.recorder = &MockDifferMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDiffer) EXPECT() *MockDifferMockRecorder {
	return m.recorder
}

// GetMissingCommits mocks base method.
func (m *MockDiffer) GetMissingCommits(ctx context.Context, repo *git.Repository, repoName *github.RepoName, since *time.Time, upstreamConfig config.Upstream) ([]*object.Commit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMissingCommits", ctx, repo, repoName, since, upstreamConfig)
	ret0, _ := ret[0].([]*object.Commit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMissingCommits indicates an expected call of GetMissingCommits.
func (mr *MockDifferMockRecorder) GetMissingCommits(ctx, repo, repoName, since, upstreamConfig interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMissingCommits", reflect.TypeOf((*MockDiffer)(nil).GetMissingCommits), ctx, repo, repoName, since, upstreamConfig)
}
