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

	// :integer rejects minimumSignificantDigits, minimumFractionDigits, maximumFractionDigits, roundingIncrement
	for _, opt := range []string{
		"minimumSignificantDigits",
		"minimumFractionDigits",
		"maximumFractionDigits",
		"roundingIncrement",
	} {
		_, err = integerFunc(NewResolvedValue(4), Options{opt: NewResolvedValue(2)}, language.AmericanEnglish)
		if !errors.Is(err, mf2.ErrBadOption) {
			t.Errorf("want ErrBadOption for :integer with %s, got %v", opt, err)
		}
	}
}

func Test_Number_RoundingIncrement(t *testing.T) {
	t.Parallel()

	// decimal
	assert := assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 5, "minimumFractionDigits": 2, "maximumFractionDigits": 2},
		language.AmericanEnglish)
	assert(1.23, "1.25")
	assert(1.21, "1.20")
	assert(1.125, "1.15")

	assert = assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 5, "minimumFractionDigits": 2},
		language.AmericanEnglish)
	assert(1.23, "1.25")

	assert = assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 5},
		language.AmericanEnglish)
	assert(12, "10")
	assert(13, "15")

	assert = assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 25},
		language.AmericanEnglish)
	assert(37, "25")
	assert(38, "50")

	assert = assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 50, "minimumFractionDigits": 2, "maximumFractionDigits": 2},
		language.AmericanEnglish)
	assert(1.23, "1.00")
	assert(1.26, "1.50")
	assert(-1.25, "-1.50")
	assert(-1.24, "-1.00")

	assert = assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 100, "minimumFractionDigits": 2, "maximumFractionDigits": 2},
		language.AmericanEnglish)
	assert(1.23, "1.00")
	assert(1.50, "2.00")

	assert = assertFormat(t, numberFunc,
		map[string]any{"roundingIncrement": 100, "minimumFractionDigits": 1, "maximumFractionDigits": 1},
		language.AmericanEnglish)
	assert(12.3, "10.0")
	assert(15.0, "20.0")

	assert = assertFormat(t, numberFunc,
		map[string]any{
			"roundingIncrement":     5,
			"minimumFractionDigits": 2,
			"maximumFractionDigits": 2,
			"signDisplay":           "exceptZero",
		},
		language.AmericanEnglish)
	assert(0.01, "0.00")
	assert(0.04, "+0.05")

	// percent
	assert = assertFormat(t, numberFunc,
		map[string]any{
			"style":                 "percent",
			"roundingIncrement":     5,
			"minimumFractionDigits": 2,
			"maximumFractionDigits": 2,
		},
		language.AmericanEnglish)
	assert(0.1234, "12.35%")
	assert(0.1237, "12.35%")

	assert = assertFormat(t, numberFunc,
		map[string]any{"style": "percent", "roundingIncrement": 5},
		language.AmericanEnglish)
	assert(0.12, "10%")
	assert(0.13, "15%")
}

func Test_Number_RoundingIncrement_Errors(t *testing.T) {
	t.Parallel()

	for _, badInc := range []any{0, 3, -5, 7, 15, "foo"} {
		opts := Options{"roundingIncrement": NewResolvedValue(badInc)}

		_, err := numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
		if !errors.Is(err, mf2.ErrBadOption) {
			t.Errorf("want ErrBadOption for roundingIncrement %v, got %v", badInc, err)
		}
	}

	// roundingIncrement incompatible with significant digits
	opts := Options{
		"roundingIncrement":        NewResolvedValue(5),
		"minimumSignificantDigits": NewResolvedValue(2),
	}

	_, err := numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
	if !errors.Is(err, mf2.ErrBadOption) {
		t.Errorf("want ErrBadOption, got %v", err)
	}

	opts = Options{
		"roundingIncrement":        NewResolvedValue(5),
		"maximumSignificantDigits": NewResolvedValue(2),
	}

	_, err = numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
	if !errors.Is(err, mf2.ErrBadOption) {
		t.Errorf("want ErrBadOption, got %v", err)
	}

	// minimumFractionDigits != maximumFractionDigits when roundingIncrement is set
	opts = Options{
		"roundingIncrement":     NewResolvedValue(5),
		"minimumFractionDigits": NewResolvedValue(1),
		"maximumFractionDigits": NewResolvedValue(2),
	}

	_, err = numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
	if !errors.Is(err, mf2.ErrBadOption) {
		t.Errorf("want ErrBadOption, got %v", err)
	}

	opts = Options{
		"roundingIncrement":     NewResolvedValue(5),
		"maximumFractionDigits": NewResolvedValue(2),
	}

	_, err = numberFunc(NewResolvedValue(4.2), opts, language.AmericanEnglish)
	if !errors.Is(err, mf2.ErrBadOption) {
		t.Errorf("want ErrBadOption, got %v", err)
	}
}

func Test_Number_RoundingIncrement_Selection(t *testing.T) {
	t.Parallel()

	// exact match with roundingIncrement
	v, err := numberFunc(NewResolvedValue(1.23),
		Options{
			"roundingIncrement":     NewResolvedValue(5),
			"minimumFractionDigits": NewResolvedValue(2),
			"maximumFractionDigits": NewResolvedValue(2),
		}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := v.selectKey([]string{"1.25", "other"}); got != "1.25" {
		t.Errorf("want '1.25', got '%s'", got)
	}

	// plural selection with roundingIncrement in French: 1.7 rounds to 1.5 ("one"), 1.9 rounds to 2.0 ("other")
	v, err = numberFunc(NewResolvedValue(1.7),
		Options{
			"roundingIncrement":     NewResolvedValue(5),
			"minimumFractionDigits": NewResolvedValue(1),
			"maximumFractionDigits": NewResolvedValue(1),
		}, language.French)
	if err != nil {
		t.Fatal(err)
	}

	if got := v.selectKey([]string{"one", "other"}); got != "one" {
		t.Errorf("want 'one', got '%s'", got)
	}

	v, err = numberFunc(NewResolvedValue(1.9),
		Options{
			"roundingIncrement":     NewResolvedValue(5),
			"minimumFractionDigits": NewResolvedValue(1),
			"maximumFractionDigits": NewResolvedValue(1),
		}, language.French)
	if err != nil {
		t.Fatal(err)
	}

	if got := v.selectKey([]string{"one", "other"}); got != "other" {
		t.Errorf("want 'other', got '%s'", got)
	}

	// operand with roundingIncrement passed to :number should retain it
	numVal, err := numberFunc(NewResolvedValue(1.23),
		Options{
			"roundingIncrement":     NewResolvedValue(5),
			"minimumFractionDigits": NewResolvedValue(2),
			"maximumFractionDigits": NewResolvedValue(2),
		}, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	numVal2, err := numberFunc(numVal, nil, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := numVal2.format(); got != "1.25" {
		t.Errorf("want '1.25', got '%s'", got)
	}

	// operand with roundingIncrement passed to :integer should discard it
	intVal, err := integerFunc(numVal, nil, language.AmericanEnglish)
	if err != nil {
		t.Fatal(err)
	}

	if got := intVal.format(); got != "1" {
		t.Errorf("want '1', got '%s'", got)
	}
}
