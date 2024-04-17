package template

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func Test_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       any
		options     map[string]any
		expected    string
		expectedErr bool
	}{
		// positive
		{
			name:     "int",
			input:    53,
			options:  nil,
			expected: "53",
		},
		{
			name:     "date",
			input:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			options:  nil,
			expected: "2021-01-01 00:00:00 +0000 UTC",
		},
		// negative
		{
			name:        "illegal type", // does not implement stringer, and is not castable to string
			input:       struct{}{},
			options:     nil,
			expectedErr: true,
		},
		{
			name:        "illegal options", // string function does not support any options
			input:       2,
			options:     map[string]any{"will": "fail"},
			expectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := stringRegistryFunc.Format(test.input, test.options, language.AmericanEnglish)

			if test.expectedErr {
				require.Error(t, err)
				require.Empty(t, actual)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}
