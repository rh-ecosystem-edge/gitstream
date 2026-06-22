package github

import (
	"errors"
	"os"

	"github.com/rh-ecosystem-edge/gitstream/internal/process"
)

type Commit struct {
	Message string
	SHA     string
}

type BaseData struct {
	AppName     string
	Commit      Commit
	JobID       string
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

// GetJobID returns the Job ID from environment variables.
// It first tries GITHUB_RUN_ID (GitHub Actions), then falls back to JOB_ID (generic CI).
// Returns empty string if no Job ID is found.
func GetJobID() string {
	if jobID, exists := os.LookupEnv("GITHUB_RUN_ID"); exists && jobID != "" {
		return jobID
	}
	if jobID, exists := os.LookupEnv("JOB_ID"); exists && jobID != "" {
		return jobID
	}
	return ""
}
