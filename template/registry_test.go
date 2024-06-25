package template

import (
	"testing"

	"golang.org/x/text/language"
)

func assertFormat(t *testing.T, f RegistryFunc, options map[string]any, locale language.Tag) func(in any, out string) {
	t.Helper()

	return func(in any, out string) {
		result, err := f.Format(in, options, locale)
		if err != nil {
			t.Error(err)
		}

		if out != result {
			t.Errorf("want %s, got %s", out, result)
		}
	}
}
