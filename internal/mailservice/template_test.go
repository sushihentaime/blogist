package mailservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTemplate(t *testing.T) {
	template := &Template{}

	testCases := []struct {
		name         string
		templateName string
		data         any
		expectedErr  bool
	}{
		{
			name:         "success",
			templateName: "activation_email.html",
			data: struct {
				ActivationLink string
				LinkName       string
			}{
				ActivationLink: "http://localhost:8080/activate?token=123",
				LinkName:       "Activate Account",
			},
			expectedErr: false,
		},
		{
			name:         "invalid template name",
			templateName: "invalid_template.html",
			data:         nil,
			expectedErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, p, h, err := template.ParseTemplate(tc.templateName, tc.data)
			assert.Equal(t, tc.expectedErr, err != nil)

			if err == nil {
				assert.NotEmpty(t, s.String())
				assert.NotEmpty(t, p.String())
				assert.NotEmpty(t, h.String())
			}
		})
	}
}
