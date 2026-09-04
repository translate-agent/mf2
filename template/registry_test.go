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

func Test_oneOf(t *testing.T) {
	t.Parallel()

	t.Run("valid and invalid options", func(t *testing.T) {
		t.Parallel()

		validator := oneOf("a", "b", "c")

		err := validator("a")
		if err != nil {
			t.Errorf("want nil, got %v", err)
		}

		err = validator("d")
		if err == nil {
			t.Error("want error, got nil")
		}
	})

	t.Run("empty options panic", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("want panic for empty oneOf, got none")
			}
		}()

		_ = oneOf[string]()
	})
}
