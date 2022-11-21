package github

import (
	"errors"

	"github.com/qbarrand/gitstream/internal/process"
)

type Commit struct {
	Message string
	SHA     string
}

type BaseData struct {
	AppName     string
	Commit      Commit
	Markup      string
	UpstreamURL string
}

type IssueData struct {
	BaseData
	Error error
}

func (is *IssueData) ProcessError() *process.Error {
	pe := &process.Error{}

	if errors.As(is.Error, &pe) {
		return pe
	}

	return nil
}

type PRData BaseData
