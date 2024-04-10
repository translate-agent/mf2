package mf2_test

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.expect.digital/mf2/template"
	"golang.org/x/text/language"
)

type WgTest struct {
	// The MF2 message to be tested.
	Src string `json:"src"`
	// The locale to use for formatting. Defaults to 'en-US'.
	Locale *language.Tag `json:"locale"`
	// Parameters to pass in to the formatter for resolving external variables.
	Params map[string]any `json:"params"`
	// The expected result of formatting the message to a string.
	Expected string `json:"exp"`
	// The expected result of formatting the message to parts.
	Parts []any `json:"parts"`
	// A normalixed form of `src`, for testing stringifiers.
	CleanSrc string `json:"cleanSrc"`
	// The runtime errors expected to be emitted when formatting the message.
	Errors []struct {
		Type string `json:"type"`
	} `json:"errors"`
}

//go:embed .message-format-wg/test/syntax-errors.json
var wgSyntaxErrors []byte

func TestWgSyntaxErrors(t *testing.T) {
	t.Parallel()

	var inputs []string

	require.NoError(t, json.Unmarshal(wgSyntaxErrors, &inputs))

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			templ, err := template.New().Parse(input)
			if err != nil { // test passes, syntax error
				return
			}

			_, err = templ.Sprint(nil)
			require.Error(t, err)
		})
	}
}

//go:embed .message-format-wg/test/test-core.json
var wgCore []byte

func TestWgCore(t *testing.T) {
	t.Skip() // TODO(jhorsts): tests fail, fix in smaller PRs. Issue #60
	t.Parallel()

	var tests []WgTest

	require.NoError(t, json.Unmarshal(wgCore, &tests))

	for _, test := range tests {
		t.Run(test.Src, func(t *testing.T) {
			t.Parallel()

			assertWgTest(t, test)
		})
	}
}

//go:embed .message-format-wg/test/test-functions.json
var wgFunctions []byte

func TestWgFunctions(t *testing.T) {
	t.Skip() // TODO(jhorsts): tests fail, fix in smaller PRs.
	t.Parallel()

	var tests map[string][]WgTest

	err := json.Unmarshal(wgFunctions, &tests)
	require.NoError(t, err)

	for funcName, funcTests := range tests {
		t.Run(funcName, func(t *testing.T) {
			for _, test := range funcTests {
				t.Run(test.Src, func(t *testing.T) {
					t.Parallel()

					assertWgTest(t, test)
				})
			}
		})
	}
}

// assertWgTest asserts MF2 WG defined tests (INCOMPLETE).
func assertWgTest(t *testing.T, test WgTest) {
	t.Helper()

	var options []template.Option
	if test.Locale != nil {
		options = append(options, template.WithLocale(*test.Locale))
	}

	templ, err := template.New(options...).Parse(test.Src)
	require.NoError(t, err)

	actual, err := templ.Sprint(test.Params)

	for _, wgErr := range test.Errors {
		switch wgErr.Type {
		default:
			t.Errorf("asserting error %s is not implemented", wgErr)
		case "missing-func":
			require.ErrorIs(t, err, template.ErrUnknownFunction)
		case "not-selectable":
			require.ErrorIs(t, err, template.ErrSelection)
		case "unresolved-var":
			require.ErrorIs(t, err, template.ErrUnresolvedVariable)
		case "unsupported-statement":
			require.ErrorIs(t, err, template.ErrUnsupportedStatement)
		case "unsupported-annotation":
			require.ErrorIs(t, err, template.ErrUnsupportedExpression)
		}
	}

	assert.Equal(t, test.Expected, actual)
}
