package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func assertFormat(t *testing.T, f RegistryFunc, options map[string]any, locale language.Tag) func(in any, out string) {
	t.Helper()

	return func(in any, out string) {
		result, err := f.Format(in, options, locale)

		require.NoError(t, err)
		assert.Equal(t, out, result)
	}
}
