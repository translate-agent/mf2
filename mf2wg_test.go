package mf2_test

import (
	"cmp"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.expect.digital/mf2"
	"go.expect.digital/mf2/template"
	"golang.org/x/text/language"
)

var failing []string

func init() {
	//nolint:lll
	failing = []string{
		"TestMF2WG/Currency_function/.local_$n_=_{42_:integer}_{{{$n_:currency_currency=EUR}}}",
		"TestMF2WG/Currency_function/.local_$n_=_{42_:number}_{{{$n_:currency_currency=EUR}}}",
		"TestMF2WG/Currency_function/.local_$n_=_{42_:number}_{{{$n_:currency}}}",
		"TestMF2WG/Currency_function/.local_$n_=_{42_:currency_currency=EUR}_{{{$n_:currency}}}",
		"TestMF2WG/Currency_function/{foo_:currency}",
		"TestMF2WG/Currency_function/{:currency}",
		"TestMF2WG/Currency_function/{42_:currency}",
		"TestMF2WG/Currency_function/{$x_:currency_currency=EUR}",
		"TestMF2WG/Currency_function/{42_:currency_currency=EUR_fractionDigits=2}",
		"TestMF2WG/Currency_function/{42_:currency_currency=EUR_fractionDigits=auto}",
		"TestMF2WG/Currency_function/{42_:currency_currency=EUR}",

		"TestMF2WG/Integer_function/.local_$bad_=_{exact}_{{variable_select_{1_:integer_select=$bad}}}",
		"TestMF2WG/Integer_function/.local_$sel_=_{1_:integer_select=exact}_.local_$bad_=_{$sel_:integer}_.match_$bad_1_{{ONE}}_*_{{operand_select_{$bad}}}",
		"TestMF2WG/Integer_function/.local_$sel_=_{1_:integer_select=$bad}_.match_$sel_1_{{ONE}}_*_{{variable_select_{$sel}}}",
		"TestMF2WG/Integer_function/.local_$x_=_{1.25_:integer}_.local_$y_=_{$x_:number}_{{{$y}}}",
		"TestMF2WG/Integer_function/variable_select_{1_:integer_select=$bad}",

		"TestMF2WG/Offset_function/.local_$x_=_{52_:number_signDisplay=always}_{{{$x_:offset_subtract=10}}}",
		"TestMF2WG/Offset_function/.local_$x_=_{1_:offset_add=1}_.match_$x_1_{{=1}}_2_{{=2}}_*_{{other}}",
		"TestMF2WG/Offset_function/{:offset_add=13}",
		"TestMF2WG/Offset_function/{52_:offset_subtract=10}",
		"TestMF2WG/Offset_function/{$x_:offset_subtract=10}",
		"TestMF2WG/Offset_function/{$x_:offset_add=1}",
		"TestMF2WG/Offset_function/.local_$x_=_{10_:integer}_.local_$y_=_{$x_:offset_subtract=6}_.match_$y_10_{{=10}}_4_{{=4}}_*_{{other}}",
		"TestMF2WG/Offset_function/.local_$x_=_{41_:integer_signDisplay=always}_{{{$x_:offset_add=1}}}",
		"TestMF2WG/Offset_function/{42_:offset}",
		"TestMF2WG/Offset_function/{foo_:offset_add=13}",
		"TestMF2WG/Offset_function/{42_:offset_subtract=foo}",
		"TestMF2WG/Offset_function/{41_:offset_add=1_foo=13}",
		"TestMF2WG/Offset_function/{42_:offset_add=foo}",
		"TestMF2WG/Offset_function/{42_:offset_foo=13}",
		"TestMF2WG/Offset_function/{42_:offset_add=13_subtract=13}",
		"TestMF2WG/Offset_function/{41_:offset_add=1}",

		"TestMF2WG/Percent_function/{:percent}",
		"TestMF2WG/Percent_function/{0.12345678_:percent_maximumFractionDigits=1}",
		"TestMF2WG/Percent_function/.local_$n_=_{0.42_:number}_{{{$n_:percent}}}",
		"TestMF2WG/Percent_function/.local_$n_=_{0.01_:percent}_{{{$n_:percent}}}",
		"TestMF2WG/Percent_function/{$x_:percent}",
		"TestMF2WG/Percent_function/.local_$n_=_{42_:integer}_{{{$n_:percent}}}",
		"TestMF2WG/Percent_function/.input_{$n_:percent}_.match_$n_one_{{one}}_*_{{other}}#01",
		"TestMF2WG/Percent_function/{0.12_:percent_minimumSignificantDigits=1}",
		"TestMF2WG/Percent_function/{foo_:percent}",
		"TestMF2WG/Percent_function/.input_{$n_:percent}_.match_$n_one_{{one}}_*_{{other}}",
		"TestMF2WG/Percent_function/{1_:percent}",
		"TestMF2WG/Percent_function/{0.12345678_:percent}",
		"TestMF2WG/Percent_function/{0.12_:percent_minimumFractionDigits=1}",

		"TestMF2WG/Number_function/.local_$foo_=_{$bar_:number}_{{bar_{$foo}}}#01",
		"TestMF2WG/Number_function/.local_$bad_=_{exact}_{{variable_select_{1_:number_select=$bad}}}",
		"TestMF2WG/Number_function/.local_$sel_=_{1_:number_select=exact}_.local_$bad_=_{$sel_:number}_.match_$bad_1_{{ONE}}_*_{{operand_select_{$bad}}}",
		"TestMF2WG/Number_function/.local_$sel_=_{1_:number_select=$bad}_.match_$sel_1_{{ONE}}_*_{{variable_select_{$sel}}}",
		"TestMF2WG/Number_function/variable_select_{1_:number_select=$bad}",

		"TestMF2WG/u:_Options/أهلاً_{بالعالم_:string}",
		"TestMF2WG/u:_Options/hello_{world_:string_u:dir=auto}",
		"TestMF2WG/u:_Options/أهلاً_{world_:string_u:dir=ltr}",
		"TestMF2WG/u:_Options/أهلاً_{بالعالم_:string_u:dir=auto}",
		"TestMF2WG/u:_Options/أهلاً_{بالعالم_:string_u:dir=rtl}",
		"TestMF2WG/u:_Options/hello_{world_:string_u:dir=ltr_u:id=foo}",
		"TestMF2WG/u:_Options/hello_{4.2_:number_u:locale=fr}",
		"TestMF2WG/u:_Options/{#tag_u:dir=rtl_u:locale=ar}content{/ns:tag}",
		"TestMF2WG/u:_Options/{#tag_u:dir=rtl}content{/ns:tag}",
		"TestMF2WG/u:_Options/hello_{world_:string_u:dir=rtl}",
		"TestMF2WG/u:_Options/.local_$world_=_{world_:string_u:dir=ltr_u:id=foo}_{{hello_{$world}}}",

		"TestMF2WG/Bidi_support/.local_$\\u200efoo\\u200f_=_{3}_{{{$\\u200efoo\\u200f}}}",
		"TestMF2WG/Bidi_support/.local_$\\u200efoo\\u200f_=_{5}_{{{$foo}}}",
		"TestMF2WG/Bidi_support/.local_$\\u061cfoo_=_{1}_{{_{$\\u061cfoo}_}}",
		"TestMF2WG/Bidi_support/.local_$foo_=_{4}_{{{$\\u200efoo\\u200f}}}",
		"TestMF2WG/Bidi_support/.local_$x_=_{1}_{{_{\\u200e_$x_\\u200f}_}}",
		"TestMF2WG/Bidi_support/.local_$x_=_{1_:number}.match_$x\\u061c1_{{one}}_*_{{other}}",
		"TestMF2WG/Bidi_support/.local_$x_=_{1_:number}.match_$x\\u061c1_{{one}}*_{{other}}",
		"TestMF2WG/Bidi_support/.local_$x_=_{1}_\\u200f_{{_{$x}}}",
		"TestMF2WG/Bidi_support/.local_$x_=_{1}_{{_{$x}}}_\\u2066",
		"TestMF2WG/Bidi_support/{\\u200e_hello_\\u200f}",
		"TestMF2WG/Bidi_support/\\u200e_.local_$x_=_{1}_{{_{$x}}}",
	}
}

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

		f, err := os.Open(path)
		if err != nil {
			t.Error(err)
		}

		defer f.Close()

		var tests Tests

		err = json.NewDecoder(f).Decode(&tests)
		if err != nil {
			t.Error(err)
		}

		t.Run(tests.Scenario, func(t *testing.T) {
			t.Parallel()

			for _, test := range tests.Tests {
				t.Run(test.Src, func(t *testing.T) {
					t.Parallel()

					if slices.Contains(failing, t.Name()) {
						t.Skip()
					}

					run(t, test.Apply(tests.DefaultTestProperties))
				})
			}
		})

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// run runs single test by Message Format Working Group.
func run(t *testing.T, test Test) {
	options := []template.Option{
		template.WithFuncs(map[string]template.Func{
			"test:function": template.RegistryTestFunc("function"),
			"test:select":   template.RegistryTestFunc("select"),
			"test:format":   template.RegistryTestFunc("format"),
		}),
	}

	if test.Locale != nil {
		options = append(options, template.WithLocale(*test.Locale))
	}

	t.Log(cmp.Or(test.Description, "no test description"))

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

	got, err := templ.Sprint(input)

	// Look at the description of first assertWgErr() in this func.
	assertErr(t, test.ExpErrors, err)

	// Expected is optional. The built-in formatters is implementation
	// specific across programming languages and libraries.
	if test.Exp != nil && *test.Exp != got {
		t.Errorf("want '%s', got '%s'", *test.Exp, got)
	}
}

// Tests contains harness tests by MF2 WG, schema defined in
// ".message-format-wg/spec/schemas/v0/tests.schema.json".
type Tests struct {
	Scenario              string                `json:"scenario"`
	Description           string                `json:"description"`
	Tests                 []Test                `json:"tests"`
	DefaultTestProperties DefaultTestProperties `json:"defaultTestProperties"`
}

type Test struct {
	// The MF2 message to be tested.
	Src string `json:"src"`
	// The bidi isolation strategy.
	BidiIsolation string `json:"bidiIsolation"`
	// Information about the test scenario.
	Description string `json:"description"`
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
	ExpErrors []Error `json:"expErrors"`
	// List of features that the test relies on.
	Tags []string `json:"tags"`
	// The expected result of formatting the message to parts.
	ExpParts []map[string]any `json:"expParts"`
}

// Apply applies default properties to the test.
func (t Test) Apply(defaultProperties DefaultTestProperties) Test {
	if len(t.ExpErrors) == 0 {
		t.ExpErrors = defaultProperties.ExpErrors
	}

	if t.Locale == nil {
		t.Locale = defaultProperties.Locale
	}

	return t
}

type DefaultTestProperties struct {
	Locale    *language.Tag `json:"locale"`
	ExpErrors []Error       `json:"expErrors"`
}

type Error struct {
	Type string `json:"type"`
}

type Var struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
	Type  string `json:"type"`
}

func assertErr(t *testing.T, want []Error, err error) {
	if len(want) == 0 && err != nil {
		t.Errorf("want no error, got '%s'", err)
	}

	wantErr := func(want error) {
		if !errors.Is(err, want) {
			t.Errorf("want error '%s', got '%s'", want, err)
		}
	}

	for _, v := range want {
		switch v.Type {
		default:
			t.Errorf("asserting error '%s' is not implemented", v)
		case "bad-operand":
			wantErr(mf2.ErrBadOperand)
		case "bad-option":
			wantErr(mf2.ErrBadOption)
		case "bad-selector":
			wantErr(mf2.ErrBadSelector)
		case "duplicate-declaration":
			wantErr(mf2.ErrDuplicateDeclaration)
		case "duplicate-option-name":
			wantErr(mf2.ErrDuplicateOptionName)
		case "duplicate-variant":
			wantErr(mf2.ErrDuplicateVariant)
		case "missing-fallback-variant":
			wantErr(mf2.ErrMissingFallbackVariant)
		case "missing-selector-annotation":
			wantErr(mf2.ErrMissingSelectorAnnotation)
		case "syntax-error":
			wantErr(mf2.ErrSyntax)
		case "unknown-function":
			wantErr(mf2.ErrUnknownFunction)
		case "unresolved-variable":
			wantErr(mf2.ErrUnresolvedVariable)
		case "unsupported-expression":
			wantErr(mf2.ErrUnsupportedExpression)
		case "unsupported-statement":
			wantErr(mf2.ErrUnsupportedStatement)
		case "variant-key-mismatch":
			wantErr(mf2.ErrVariantKeyMismatch)
		}
	}
}
