package mf2_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"go.expect.digital/mf2"
	"go.expect.digital/mf2/template"
	"golang.org/x/text/language"
)

var failing []string

func init() {
	//nolint:lll
	failing = []string{
		"TestMF2WG/.message-format-wg/test/tests/fallback.json/{42_:test:function_fails=format}",
		"TestMF2WG/.message-format-wg/test/tests/fallback.json/{|C:\\\\|_:test:function_fails=format}",
		"TestMF2WG/.message-format-wg/test/tests/fallback.json/.local_$var_=_{|val|_:test:undefined}_{{{$var}}}",
		"TestMF2WG/.message-format-wg/test/tests/fallback.json/.local_$var_=_{|val|_:test:undefined}_{{{$var_:test:function}}}",

		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/.local_$n_=_{42_:integer}_{{{$n_:currency_currency=EUR}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/.local_$n_=_{42_:number}_{{{$n_:currency_currency=EUR}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/.local_$n_=_{42_:number}_{{{$n_:currency}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/.local_$n_=_{42_:currency_currency=EUR}_{{{$n_:currency}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{foo_:currency}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{:currency}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{42_:currency}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{$x_:currency_currency=EUR}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{42_:currency_currency=EUR_fractionDigits=2}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{42_:currency_currency=EUR_fractionDigits=auto}",
		"TestMF2WG/.message-format-wg/test/tests/functions/currency.json/{42_:currency_currency=EUR}",

		"TestMF2WG/.message-format-wg/test/tests/functions/datetime.json/{|2006-01-02T15:04:06|_:datetime_year=numeric_month=|2-digit|}",

		"TestMF2WG/.message-format-wg/test/tests/functions/integer.json/.local_$bad_=_{exact}_{{variable_select_{1_:integer_select=$bad}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/integer.json/.local_$sel_=_{1_:integer_select=exact}_.local_$bad_=_{$sel_:integer}_.match_$bad_1_{{ONE}}_*_{{operand_select_{$bad}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/integer.json/.local_$sel_=_{1_:integer_select=$bad}_.match_$sel_1_{{ONE}}_*_{{variable_select_{$sel}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/integer.json/.local_$x_=_{1.25_:integer}_.local_$y_=_{$x_:number}_{{{$y}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/integer.json/variable_select_{1_:integer_select=$bad}",

		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/.local_$x_=_{52_:number_signDisplay=always}_{{{$x_:offset_subtract=10}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/.local_$x_=_{1_:offset_add=1}_.match_$x_1_{{=1}}_2_{{=2}}_*_{{other}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{:offset_add=13}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{52_:offset_subtract=10}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{$x_:offset_subtract=10}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{$x_:offset_add=1}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/.local_$x_=_{10_:integer}_.local_$y_=_{$x_:offset_subtract=6}_.match_$y_10_{{=10}}_4_{{=4}}_*_{{other}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/.local_$x_=_{41_:integer_signDisplay=always}_{{{$x_:offset_add=1}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{42_:offset}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{foo_:offset_add=13}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{42_:offset_subtract=foo}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{41_:offset_add=1_foo=13}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{42_:offset_add=foo}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{42_:offset_foo=13}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{42_:offset_add=13_subtract=13}",
		"TestMF2WG/.message-format-wg/test/tests/functions/offset.json/{41_:offset_add=1}",

		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{:percent}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{0.12345678_:percent_maximumFractionDigits=1}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/.local_$n_=_{0.42_:number}_{{{$n_:percent}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/.local_$n_=_{0.01_:percent}_{{{$n_:percent}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{$x_:percent}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/.local_$n_=_{42_:integer}_{{{$n_:percent}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/.input_{$n_:percent}_.match_$n_one_{{one}}_*_{{other}}#01",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{0.12_:percent_minimumSignificantDigits=1}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{foo_:percent}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/.input_{$n_:percent}_.match_$n_one_{{one}}_*_{{other}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{1_:percent}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{0.12345678_:percent}",
		"TestMF2WG/.message-format-wg/test/tests/functions/percent.json/{0.12_:percent_minimumFractionDigits=1}",

		"TestMF2WG/.message-format-wg/test/tests/functions/number.json/.local_$foo_=_{$bar_:number}_{{bar_{$foo}}}#01",
		"TestMF2WG/.message-format-wg/test/tests/functions/number.json/.local_$bad_=_{exact}_{{variable_select_{1_:number_select=$bad}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/number.json/.local_$sel_=_{1_:number_select=exact}_.local_$bad_=_{$sel_:number}_.match_$bad_1_{{ONE}}_*_{{operand_select_{$bad}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/number.json/.local_$sel_=_{1_:number_select=$bad}_.match_$sel_1_{{ONE}}_*_{{variable_select_{$sel}}}",
		"TestMF2WG/.message-format-wg/test/tests/functions/number.json/variable_select_{1_:number_select=$bad}",

		"TestMF2WG/.message-format-wg/test/tests/u-options.json/أهلاً_{بالعالم_:string}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/hello_{world_:string_u:dir=auto}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/أهلاً_{world_:string_u:dir=ltr}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/أهلاً_{بالعالم_:string_u:dir=auto}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/أهلاً_{بالعالم_:string_u:dir=rtl}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/hello_{world_:string_u:dir=ltr_u:id=foo}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/hello_{4.2_:number_u:locale=fr}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/{#tag_u:dir=rtl_u:locale=ar}content{/ns:tag}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/{#tag_u:dir=rtl}content{/ns:tag}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/hello_{world_:string_u:dir=rtl}",
		"TestMF2WG/.message-format-wg/test/tests/u-options.json/.local_$world_=_{world_:string_u:dir=ltr_u:id=foo}_{{hello_{$world}}}",

		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$\\u200efoo\\u200f_=_{3}_{{{$\\u200efoo\\u200f}}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$\\u200efoo\\u200f_=_{5}_{{{$foo}}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$\\u061cfoo_=_{1}_{{_{$\\u061cfoo}_}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$foo_=_{4}_{{{$\\u200efoo\\u200f}}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$x_=_{1}_{{_{\\u200e_$x_\\u200f}_}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$x_=_{1_:number}.match_$x\\u061c1_{{one}}_*_{{other}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$x_=_{1_:number}.match_$x\\u061c1_{{one}}*_{{other}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$x_=_{1}_\\u200f_{{_{$x}}}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/.local_$x_=_{1}_{{_{$x}}}_\\u2066",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/{\\u200e_hello_\\u200f}",
		"TestMF2WG/.message-format-wg/test/tests/bidi.json/\\u200e_.local_$x_=_{1}_{{_{$x}}}",
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

		t.Run(path, func(t *testing.T) {
			t.Parallel()

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
			"test:select": template.RegistryTestFunc("select"),
			"test:format": template.RegistryTestFunc("format"),
		}),
	}

	if test.Locale != nil {
		options = append(options, template.WithLocale(*test.Locale))
	}

	if test.Description != "" {
		t.Log(test.Description)
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
	Tests                 []Test                `json:"tests"`
	DefaultTestProperties DefaultTestProperties `json:"defaultTestProperties"`
}

type Test struct {
	// The MF2 message to be tested.
	Src string `json:"src"`
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
		err := json.Unmarshal(data, &e.Errors)
		if err != nil {
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

func assertErr(t *testing.T, want Errors, err error) {
	// we expect error but we don't know the exact error
	if want.Expected != nil && *want.Expected && err == nil {
		t.Error("want error, got nil")
	}

	if len(want.Errors) == 0 && err != nil {
		t.Errorf("want no error, got '%s'", err)
	}

	wantErr := func(want error) {
		if !errors.Is(err, want) {
			t.Errorf("want error '%s', got '%s'", want, err)
		}
	}

	for _, v := range want.Errors {
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
