package blogservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeMarkdown(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no script tag",
			input: "Hello, World!",
			want:  "Hello, World!",
		},
		{
			name:  "script tag",
			input: "<script>alert('Hello, World!');</script>",
			want:  "",
		},
		{
			name: "multiple script tags",
			input: `Here is some text.
					<script>alert('Hello, world!');</script>
					More text.
					<SCRIPT SRC="evil.js"></SCRIPT>`,
			want: "Here is some text.\n\t\n\tMore text.\n\t",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := sanitizeMarkdown(tc.input)
			assert.Equal(t, tc.want, output)
		})
	}

}
