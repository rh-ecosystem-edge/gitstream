package github

import (
	"embed"
	"text/template"
)

var (
	//go:embed templates/*
	tmplFS embed.FS

	templates = template.Must(
		template.ParseFS(tmplFS, "templates/*.tmpl"),
	)
)
