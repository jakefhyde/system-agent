package localplan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipFile(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		skips    map[string]bool
		expected bool
	}{
		{
			name:     "default",
			fileName: "",
			skips:    map[string]bool{},
			expected: true,
		},
		{
			name:     "dot prefix",
			fileName: ".test",
			skips:    map[string]bool{},
			expected: true,
		},
		{
			name:     "plan suffix",
			fileName: "test.plan",
			skips:    map[string]bool{},
			expected: false,
		},
		{
			name:     "no skips",
			fileName: "test",
			skips:    map[string]bool{},
			expected: true,
		},
		{
			name:     "should skip",
			fileName: "test",
			skips: map[string]bool{
				"test": true,
			},
			expected: true,
		},
		{
			name:     "shouldn't skip",
			fileName: "test",
			skips: map[string]bool{
				"test": false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := skipFile(tt.fileName, tt.skips)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
