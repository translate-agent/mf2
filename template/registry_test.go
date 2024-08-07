package template

import (
	"testing"

	"golang.org/x/text/language"
)

func assertFormat(t *testing.T, f Func, options map[string]any, locale language.Tag) func(in any, want string) {
	t.Helper()

	opts := make(Options, len(options))
	for k, v := range options {
		opts[k] = NewResolvedValue(v)
	}

	return func(in any, want string) {
		v, err := f(NewResolvedValue(in), opts, locale)
		if err != nil {
			t.Error(err)

			return
		}

		result := v.format()

		if want != result {
			t.Errorf("want '%s', got '%s'", want, result)
		}
	}
}
