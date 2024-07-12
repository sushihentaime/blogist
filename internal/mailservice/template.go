package mailservice

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*
var templateFS embed.FS

func NewTemplate() *Template {
	return &Template{}
}

// ParseTemplate function that parses the email template. The data parameter should be a struct that contains the data to be used in the template.
func (tp *Template) ParseTemplate(name string, data any) (*bytes.Buffer, *bytes.Buffer, *bytes.Buffer, error) {
	t, err := template.New("email").ParseFS(templateFS, "templates/"+name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("could not parse template: %w", err)
	}

	subject := new(bytes.Buffer)
	err = t.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return nil, nil, nil, err
	}

	plainBody := new(bytes.Buffer)
	err = t.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return nil, nil, nil, err
	}

	htmlBody := new(bytes.Buffer)
	err = t.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return nil, nil, nil, err
	}

	return subject, plainBody, htmlBody, nil
}
