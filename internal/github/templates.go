package github

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

var (
	//go:embed templates/*
	tmplFS embed.FS

	templates = template.Must(
		template.ParseFS(tmplFS, "templates/*.tmpl"),
	)
)

// ExecuteAssignmentCommentTemplate executes the assignment comment template with the provided data
func ExecuteAssignmentCommentTemplate(buf *bytes.Buffer, data *AssignmentCommentData) error {
	if err := templates.ExecuteTemplate(buf, "assignment_comment.tmpl", data); err != nil {
		return fmt.Errorf("could not execute assignment comment template: %v", err)
	}
	return nil
}
