package template

import (
	"errors"
	"testing"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

func Test_Number(t *testing.T) {
	t.Parallel()

	// decimal

	assert := assertFormat(t, numberFunc, nil, language.Latvian)
	assert(-0.1234, "-0,123")
	assert(0, "0")
	assert(0.1234, "0,123")

	assert = assertFormat(t, numberFunc, map[string]any{"signDisplay": "auto"}, language.AmericanEnglish)
	assert(-0.15, "-0.15")
	assert(0, "0")
	assert(0.15, "0.15")

	assert = assertFormat(t, numberFunc, map[string]any{"signDisplay": "always"}, language.AmericanEnglish)
	assert(-0.15, "-0.15")
	assert(0, "+0")
	assert(0.15, "+0.15")

	assert = assertFormat(t, numberFunc, map[string]any{"signDisplay": "exceptZero"}, language.AmericanEnglish)
	assert(-0.15, "-0.15")
	assert(0, "0")
	assert(0.15, "+0.15")

	assert = assertFormat(t, numberFunc, map[string]any{"signDisplay": "never"}, language.AmericanEnglish)
	assert(-0.15, "0.15")
	assert(0, "0")
	assert(0.15, "0.15")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumFractionDigits": 2}, language.AmericanEnglish)
	assert(0, "0.00")

	assert = assertFormat(t, numberFunc, map[string]any{"maximumFractionDigits": 1}, language.AmericanEnglish)
	assert(0.15, "0.2")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumIntegerDigits": 3}, language.AmericanEnglish)
	assert(1, "001")

	assert = assertFormat(t, numberFunc, map[string]any{"maximumSignificantDigits": 2}, language.AmericanEnglish)
	assert(1.23, "1.2")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumSignificantDigits": 4}, language.AmericanEnglish)
	assert(4.2, "4.200")
	assert(-4.2, "-4.200")
	assert(42, "42.00")
	assert(0.042, "0.04200")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumSignificantDigits": 2}, language.AmericanEnglish)
	assert(4200, "4,200")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumSignificantDigits": 3}, language.AmericanEnglish)
	assert(0, "0.00")
	assert(0.5, "0.500")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumSignificantDigits": 1}, language.AmericanEnglish)
	assert(0, "0")

	assert = assertFormat(t, numberFunc,
		map[string]any{"minimumSignificantDigits": 2, "maximumSignificantDigits": 3}, language.AmericanEnglish)
	assert(1.2345, "1.23")
	assert(1, "1.0")
	assert(1.2, "1.2")

	assert = assertFormat(t, numberFunc, map[string]any{"minimumSignificantDigits": 5}, language.AmericanEnglish)
	assert(1.23456, "1.23456")

	assert = assertFormat(t, numberFunc, map[string]any{"maximumFractionDigits": "1"}, language.AmericanEnglish)
	assert(0.15, "0.2")

	// percent

	assert = assertFormat(t, numberFunc, map[string]any{"style": "percent"}, language.Latvian)
	assert(-0.127, "-13%")
	assert(0, "0%")
	assert(0.127, "13%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "signDisplay": "auto"}, language.AmericanEnglish)
	assert(-0.127, "-13%")
	assert(0, "0%")
	assert(0.127, "13%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "signDisplay": "always"}, language.AmericanEnglish)
	assert(-0.127, "-13%")
	assert(0, "+0%")
	assert(0.127, "+13%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "signDisplay": "exceptZero"}, language.AmericanEnglish)
	assert(-0.127, "-13%")
	assert(0, "0%")
	assert(0.127, "+13%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "signDisplay": "never"}, language.AmericanEnglish)
	assert(-0.127, "13%")
	assert(0, "0%")
	assert(0.127, "13%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "minimumFractionDigits": 2}, language.AmericanEnglish)
	assert(0, "0.00%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "maximumFractionDigits": 1}, language.Latvian)
	assert(0.1275, "12,8%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "minimumIntegerDigits": 3}, language.AmericanEnglish)
	assert(0.01, "001%")

	assert = assertFormat(t, numberFunc,
		map[string]any{
			"style":                    "percent",
			"maximumFractionDigits":    5,
			"maximumSignificantDigits": 4,
		}, language.AmericanEnglish)
	assert(0.12345, "12.34%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "minimumSignificantDigits": 1}, language.AmericanEnglish)
	assert(0.12, "12%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "minimumSignificantDigits": 4}, language.AmericanEnglish)
	assert(0.12, "12.00%")
	assert(0.12345, "12.345%")

	assert = assertFormat(t, numberFunc, map[string]any{}, language.Latvian)
	assert("0.1", "0,1")

	// bad options
	opts := Options{
		"minimumSignificantDigits": NewResolvedValue(5),
		"maximumSignificantDigits": NewResolvedValue(2),
	}

	_, err := numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
	if !errors.Is(err, mf2.ErrBadOption) {
		t.Errorf("want ErrBadOption, got %v", err)
	}

	opts = Options{
		"minimumSignificantDigits": NewResolvedValue(0),
	}

	_, err = numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
	if !errors.Is(err, mf2.ErrBadOption) {
		t.Errorf("want ErrBadOption, got %v", err)
	}

	// selection with minimumSignificantDigits
	v, err := numberFunc(NewResolvedValue(1),
		Options{"minimumSignificantDigits": NewResolvedValue(1)}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := v.selectKey([]string{"one", "other"}); got != "one" {
		t.Errorf("want 'one', got '%s'", got)
	}

	v, err = numberFunc(NewResolvedValue(1),
		Options{"minimumSignificantDigits": NewResolvedValue(2)}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := v.selectKey([]string{"one", "other"}); got != "other" {
		t.Errorf("want 'other', got '%s'", got)
	}

	// exact match with minimumSignificantDigits
	v, err = numberFunc(NewResolvedValue(4.2),
		Options{"minimumSignificantDigits": NewResolvedValue(4)}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := v.selectKey([]string{"4.200", "other"}); got != "4.200" {
		t.Errorf("want '4.200', got '%s'", got)
	}

	// operand with minimumSignificantDigits passed to :integer should discard it
	numVal, err := numberFunc(NewResolvedValue(4.2),
		Options{"minimumSignificantDigits": NewResolvedValue(4)}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	intVal, err := integerFunc(numVal, nil, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := intVal.format(); got != "4" {
		t.Errorf("want '4', got '%s'", got)
	}

	// operand with minimumSignificantDigits passed to :number should retain it
	numVal2, err := numberFunc(numVal, nil, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := numVal2.format(); got != "4.200" {
		t.Errorf("want '4.200', got '%s'", got)
	}

	// operand with minimumIntegerDigits=3 passed to :integer should retain it
	numVal, err = numberFunc(NewResolvedValue(4),
		Options{"minimumIntegerDigits": NewResolvedValue(3)}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	intVal, err = integerFunc(numVal, nil, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := intVal.format(); got != "004" {
		t.Errorf("want '004', got '%s'", got)
	}

	// :integer rejects minimumSignificantDigits, minimumFractionDigits, maximumFractionDigits
	for _, opt := range []string{"minimumSignificantDigits", "minimumFractionDigits", "maximumFractionDigits"} {
		_, err = integerFunc(NewResolvedValue(4), Options{opt: NewResolvedValue(2)}, language.AmericanEnglish)
		if !errors.Is(err, mf2.ErrBadOption) {
			t.Errorf("want ErrBadOption for :integer with %s, got %v", opt, err)
		}
	}
}
