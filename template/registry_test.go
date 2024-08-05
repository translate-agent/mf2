package template

import (
	"testing"

	"golang.org/x/text/language"
)

func assertFormat(t *testing.T, f Func, options map[string]any, locale language.Tag) func(in any, want string) {
	t.Helper()

	return func(in any, want string) {
		v, err := f(in, options, locale)
		if err != nil {
			t.Error(err)

			return
		}

		result := v.String()

		if want != result {
			t.Errorf("want '%s', got '%s'", want, result)
		}
	}
}
