package github

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetJobID(t *testing.T) {
	tests := []struct {
		name           string
		githubRunID    string
		jobID          string
		expectedResult string
	}{
		{
			name:           "GitHub Actions environment",
			githubRunID:    "12345678",
			jobID:          "",
			expectedResult: "12345678",
		},
		{
			name:           "Generic CI environment",
			githubRunID:    "",
			jobID:          "ci-job-987",
			expectedResult: "ci-job-987",
		},
		{
			name:           "GitHub Actions takes precedence",
			githubRunID:    "github-123",
			jobID:          "generic-456",
			expectedResult: "github-123",
		},
		{
			name:           "No job ID available",
			githubRunID:    "",
			jobID:          "",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv("GITHUB_RUN_ID")
			os.Unsetenv("JOB_ID")

			// Set test environment variables
			if tt.githubRunID != "" {
				os.Setenv("GITHUB_RUN_ID", tt.githubRunID)
			}
			if tt.jobID != "" {
				os.Setenv("JOB_ID", tt.jobID)
			}

			result := GetJobID()
			assert.Equal(t, tt.expectedResult, result)

			// Clean up
			os.Unsetenv("GITHUB_RUN_ID")
			os.Unsetenv("JOB_ID")
		})
	}
}