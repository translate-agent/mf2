package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func Test_Number(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input       any
		options     map[string]any
		locale      language.Tag
		expected    any
		name        string
		expectedErr bool
	}{
		// positive
		{
			name:     "int",
			input:    53,
			expected: "53",
		},
		{
			name:     "style",
			input:    0.23,
			options:  map[string]any{"style": "percent"},
			expected: "23%",
		},
		{
			name:     "style",
			input:    0.127,
			options:  map[string]any{"style": "percent"},
			locale:   language.Latvian,
			expected: "12,7%",
		},
		{
			name:     "signDisplay and percent style",
			input:    0.23,
			options:  map[string]any{"style": "percent", "signDisplay": "always"},
			expected: "+23%",
		},
		// negative
		{
			name:        "not implemented",
			input:       0.23,
			options:     map[string]any{"compactDisplay": "short"},
			expectedErr: true,
		},
		{
			name:        "illegal type",
			input:       struct{}{},
			options:     nil,
			expectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual, err := numberRegistryFunc.Format(test.input, test.options, test.locale)

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
