package mf2_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.expect.digital/mf2/template"
	"golang.org/x/text/language"
)

// TestMF2WG runs tests by Message Format Working Group.
func TestMF2WG(t *testing.T) {
	t.Parallel()

	err := filepath.Walk(".message-format-wg/test/tests", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// skip the tests that are NOT IMPLEMENTED yet fully
		if slices.Contains([]string{
			".message-format-wg/test/tests/data-model-errors.json",
			".message-format-wg/test/tests/functions/datetime.json",
			".message-format-wg/test/tests/functions/integer.json",
			".message-format-wg/test/tests/functions/string.json",
		}, path) {
			t.Skip()
		}

		t.Run(path, func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(path)

			require.NoError(t, err)

			defer f.Close()

			var tests Tests

			require.NoError(t, json.NewDecoder(f).Decode(&tests))

			for _, test := range tests.Tests {
				t.Run(test.Src, func(t *testing.T) {
					t.Parallel()

					run(t, test.Apply(tests.DefaultTestProperties))
				})
			}
		})

		return nil
	})

	require.NoError(t, err)
}

// run runs single test by Message Format Working Group.
func run(t *testing.T, test Test) {
	var options []template.Option
	if test.Locale != nil {
		options = append(options, template.WithLocale(*test.Locale))
	}

	templ, err := template.New(options...).Parse(test.Src)
	// The implementation returns error in two places:
	// - when parsing the template
	// - when executing the template
	//
	// This affects asserting the error. We do not know if expected error is parsing
	// or executing the template.
	//
	// If the test expects parse error but parsing does not return error,
	// then the test tries to assert when executing error.
	if err != nil {
		assertErr(t, test.ExpErrors, err)

		return
	}

	input := make(map[string]interface{}, len(test.Params))

	for _, v := range test.Params {
		input[v.Name] = v.Value
	}

	actual, err := templ.Sprint(input)

	// Look at the description of first assertWgErr() in this func.
	assertErr(t, test.ExpErrors, err)

	// Expected is optional. The built-in formatters is implementation
	// specific across programming languages and libraries.
	if test.Exp != nil {
		assert.Equal(t, *test.Exp, actual)
	}
}

// Tests contains harness tests by MF2 WG, schema defined in
// ".message-format-wg/spec/schemas/v0/tests.schema.json".
type Tests struct {
	Tests                 []Test                `json:"tests"`
	DefaultTestProperties DefaultTestProperties `json:"defaultTestProperties"`
}

type Test struct {
	// The MF2 message to be tested.
	Src string `json:"src"`
	// The locale to use for formatting. Defaults to 'en-US'.
	Locale *language.Tag `json:"locale"`
	// Parameters to pass in to the formatter for resolving external variables.
	Params []Var `json:"params"`
	// The expected result of formatting the message to a string.
	Exp *string `json:"exp"`
	// The expected result of formatting the message to parts.
	Parts []any `json:"parts"`
	// A normalixed form of `src`, for testing stringifiers.
	CleanSrc string `json:"cleanSrc"`
	// The runtime errors expected to be emitted when formatting the message.
	ExpErrors Errors `json:"expErrors"`
}

// Apply applies default properties to the test.
func (t Test) Apply(defaultProperties DefaultTestProperties) Test {
	if t.ExpErrors.Expected == nil {
		t.ExpErrors.Expected = defaultProperties.ExpErrors.Expected
	}

	if len(t.ExpErrors.Errors) == 0 {
		t.ExpErrors.Errors = defaultProperties.ExpErrors.Errors
	}

	if t.Locale == nil {
		t.Locale = defaultProperties.Locale
	}

	return t
}

type Errors struct {
	Expected *bool
	Errors   []Error
}

func (e *Errors) UnmarshalJSON(data []byte) error {
	switch {
	default: // parse bool
		v, err := strconv.ParseBool(string(data))
		if err != nil {
			return fmt.Errorf("want bool: %w", err)
		}

		e.Expected = &v

		return nil
	case len(data) == 0:
		return nil
	case data[0] == '[': // parse errors slice
		if err := json.Unmarshal(data, &e.Errors); err != nil {
			return fmt.Errorf("want slice: %w", err)
		}

		return nil
	}
}

type DefaultTestProperties struct {
	Locale    *language.Tag `json:"locale"`
	ExpErrors Errors        `json:"expErrors"`
}

type Error struct {
	Type string `json:"type"`
}

type Var struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
	Type  string `json:"type"`
}

func assertErr(t *testing.T, expected Errors, err error) {
	if expected.Expected != nil && *expected.Expected {
		// we expect error but we don't know the exact error
		assert.NotNil(t, err) //nolint:testifylint

		return
	}

	if len(expected.Errors) == 0 {
		require.NoError(t, err)
	}

	for _, v := range expected.Errors {
		switch v.Type {
		default:
			t.Errorf("asserting error %s is not implemented", v)
		case "bad-input":
			require.ErrorIs(t, err, template.ErrOperandMismatch)
		case "missing-func":
			require.ErrorIs(t, err, template.ErrUnknownFunction)
		case "not-selectable":
			require.ErrorIs(t, err, template.ErrSelection)
		case "unresolved-variable":
			require.ErrorIs(t, err, template.ErrUnresolvedVariable)
		case "unsupported-statement":
			require.ErrorIs(t, err, template.ErrUnsupportedStatement)
		case "unsupported-expression":
			require.ErrorIs(t, err, template.ErrUnsupportedExpression)
		case "syntax-error":
			require.ErrorIs(t, err, template.ErrSyntax)
		}
	}
}
