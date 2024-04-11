package registry

import (
	"errors"
	"fmt"
	"reflect"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// https://github.com/unicode-org/message-format-wg/blob/20a61b4af534acb7ecb68a3812ca0143b34dfc76/spec/registry.xml#L147

// numberRegistryFunc is the implementation of the number function. Locale-sensitive number formatting.
var numberRegistryFunc = &Func{
	Name:           "number",
	Func:           numberFunc,
	MatchSignature: nil, // Not allowed to use in matching context
	FormatSignature: &Signature{
		IsInputRequired: true,
		ValidateInput: func(a any) error {
			if _, err := castAs[float64](a); err != nil {
				return fmt.Errorf("unsupported type: %T: %w", a, err)
			}

			return nil
		},
		Options: Options{
			{
				// Only used when notation is "compact".
				Name:           "compactDisplay",
				PossibleValues: []any{"short", "long"},
				Default:        "short",
			},
			{
				// The currency to use in currency formatting.
				// Possible values are the ISO 4217 currency codes, such as "USD" for the US dollar,
				// "EUR" for the euro, or "CNY" for the Chinese RMB — see the
				// Current currency &amp; funds code list
				// (https://www.unicode.org/cldr/charts/latest/supplemental/detailed_territory_currency_information.html).
				// There is no default value; if the style is "currency", the currency property must be provided.
				Name: "currency",
				ValidateValue: func(a any) error {
					unit, ok := a.(currency.Unit)
					if !ok {
						return fmt.Errorf("expected currency.Unit got %T", a)
					}

					var zeroVal currency.Unit
					if unit == zeroVal {
						return errors.New("currency is not set")
					}

					return nil
				},
			},
			{
				// How to display the currency in currency formatting.
				Name:           "currencyDisplay",
				PossibleValues: []any{"code", "symbol", "narrowSymbol", "name"},
			},
			{
				// In many locales, accounting format means to wrap the number with parentheses
				// instead of appending a minus sign. You can enable this formatting by setting the
				// currencySign option to "accounting".
				Name:           "currencySign",
				PossibleValues: []any{"standard", "accounting"},
				Default:        "standard",
			},
			{
				// The formatting that should be displayed for the number.
				Name:           "notation",
				PossibleValues: []any{"standard", "scientific", "engineering", "compact"},
				Default:        "standard",
			},
			{
				// Numbering system to use.
				Name: "numberingSystem",
				PossibleValues: []any{
					"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru",
					"hanidec", "khmr", "knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec",
					"telu", "thai", "tibt",
				},
			},
			{
				// When to display the sign for the number. "negative" value is Experimental.
				Name:           "signDisplay",
				PossibleValues: []any{"auto", "always", "exceptZero", "negative", "never"},
				Default:        "auto",
			},
			{
				// The formatting style to use.
				Name:           "style",
				PossibleValues: []any{"decimal", "currency", "percent", "unit"},
				Default:        "decimal",
			},
			{
				// The unit to use in unit formatting.
				// Possible values are core unit identifiers, defined in UTS #35, Part 2, Section 6.
				// A subset of units from the full list was selected for use in ECMAScript.
				// Pairs of simple units can be concatenated with "-per-" to make a compound unit.
				// There is no default value; if the style is "unit", the unit property must be provided.
				Name:          "unit",
				ValidateValue: isPositiveInteger,
			},
			{
				// The unit formatting style to use in unit formatting.
				Name:           "unitDisplay",
				PossibleValues: []any{"long", "short", "narrow"},
				Default:        "short",
			},
			{
				// The minimum number of integer digits to use.
				// A value with a smaller number of integer digits than this number will be
				// left-padded with zeros (to the specified length) when formatted.
				Name:          "minimumIntegerDigits",
				ValidateValue: isPositiveInteger,
				Default:       1,
			},
			{
				// The minimum number of fraction digits to use.
				// The default for plain number and percent formatting is 0;
				// the default for currency formatting is the number of minor unit digits provided by
				// the ISO 4217 currency code list (2 if the list doesn't provide that information).
				Name:          "minimumFractionDigits",
				ValidateValue: isPositiveInteger,
			},
			{
				// The maximum number of fraction digits to use.
				// The default for plain number formatting is the larger of minimumFractionDigits and 3;
				// the default for currency formatting is the larger of minimumFractionDigits and the number of
				// minor
				// unit digits provided by the ISO 4217 currency code list (2 if the list doesn't provide that
				// information);
				// the default for percent formatting is the larger of minimumFractionDigits and 0.
				Name:          "maximumFractionDigits",
				ValidateValue: isPositiveInteger,
			},
			{
				// The minimum number of significant digits to use.
				Name:          "minimumSignificantDigits",
				ValidateValue: isPositiveInteger,
				Default:       1,
			},
			{
				// The maximum number of significant digits to use.
				Name:          "maximumSignificantDigits",
				ValidateValue: isPositiveInteger,
				Default:       21, //nolint:gomnd
			},
		},
	},
}

// TODO: supports only style and signDisplay options.
func numberFunc(input any, options map[string]any, locale language.Tag) (any, error) {
	num, err := castAs[float64](input)
	if err != nil {
		return nil, fmt.Errorf("convert input to float64: %w", err)
	}

	for optName := range options {
		switch optName {
		case "compactDisplay", "currency", "currencyDisplay", "currencySign", "notation", "numberingSystem",
			"unit", "unitDisplay", "minimumIntegerDigits", "minimumFractionDigits",
			"maximumFractionDigits", "minimumSignificantDigits", "maximumSignificantDigits":
			return nil, fmt.Errorf("option '%s' is not implemented", optName)
		}
	}

	var (
		result string
		style  any = "decimal"
	)

	if s, ok := options["style"]; ok {
		style = s
	}

	p := message.NewPrinter(locale)

	switch style {
	case "decimal":
		result = p.Sprint(number.Decimal(num))
	case "percent":
		result = p.Sprint(number.Percent(num))
	default:
		return nil, fmt.Errorf("option '%s' is not implemented", style)
	}

	if signDisplay, ok := options["signDisplay"].(string); ok {
		switch signDisplay {
		case "auto":
		case "negative":
		case "always":
			if num >= 0 {
				result = "+" + result
			}
		case "exceptZero":
			if num > 0 {
				result = "+" + result
			}
		case "never":
			if num < 0 {
				result = result[1:]
			}
		}
	}

	return result, nil
}

// helpers

// castAs tries to cast any value to the given type.
func castAs[T any](val any) (T, error) {
	var zeroVal T
	typ := reflect.TypeOf(zeroVal)

	v := (reflect.ValueOf(val))
	if !v.Type().ConvertibleTo(typ) {
		return zeroVal, fmt.Errorf("cannot convert %v to %T", v.Type(), zeroVal)
	}

	v = v.Convert(typ)

	return v.Interface().(T), nil //nolint:forcetypeassert
}

func isPositiveInteger(v any) error {
	val, err := castAs[int](v)
	if err != nil {
		return fmt.Errorf("convert val to int: %w", err)
	}

	if val < 0 {
		return errors.New("value must be at least 0")
	}

	return nil
}
