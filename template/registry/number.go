package registry

import (
	"fmt"
	"reflect"
	"strconv"

	"golang.org/x/text/currency"
)

// https://github.com/unicode-org/message-format-wg/blob/122e64c2482b54b6eff4563120915e0f86de8e4d/spec/registry.xml#L147

var numberRegistryF = &Func{
	Name:           "number",
	Description:    "Locale-sensitive number formatting",
	Fn:             numberF,
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
				Name:           "compactDisplay",
				Description:    `Only used when notation is "compact".`,
				PossibleValues: []any{"short", "long"},
				Default:        "short",
			},
			{
				Name: "currency",
				Description: `The currency to use in currency formatting.
Possible values are the ISO 4217 currency codes, such as "USD" for the US dollar,
"EUR" for the euro, or "CNY" for the Chinese RMB — see the
Current currency &amp; funds code list
(https://www.unicode.org/cldr/charts/latest/supplemental/detailed_territory_currency_information.html).
There is no default value; if the style is "currency", the currency property must be provided.`,
				ValidateValue: func(a any) error {
					unit, ok := a.(currency.Unit)
					if !ok {
						return fmt.Errorf("expected currency.Unit got %T", a)
					}

					var zeroVal currency.Unit
					if unit == zeroVal {
						return fmt.Errorf("currency is not set")
					}

					return nil
				},
			},
			{
				Name:           "currencyDisplay",
				Description:    `How to display the currency in currency formatting.`,
				PossibleValues: []any{"code", "symbol", "narrowSymbol", "name"},
			},
			{
				Name: "currencySign",
				Description: `In many locales, accounting format means to wrap the number with parentheses
instead of appending a minus sign. You can enable this formatting by setting the
currencySign option to "accounting".`,
				PossibleValues: []any{"standard", "accounting"},
				Default:        "standard",
			},
			{
				Name:           "notation",
				Description:    "The formatting that should be displayed for the number.",
				PossibleValues: []any{"standard", "scientific", "engineering", "compact"},
				Default:        "standard",
			},
			{
				Name:        "numberingSystem",
				Description: "Numbering system to use.",
				PossibleValues: []any{
					"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru",
					"hanidec", "khmr", "knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec",
					"telu", "thai", "tibt",
				},
			},
			{
				Name:           "signDisplay",
				Description:    `When to display the sign for the number. "negative" value is Experimental.`,
				PossibleValues: []any{"auto", "always", "exceptZero", "negative", "never"},
				Default:        "auto",
			},
			{
				Name:           "style",
				Description:    "The formatting style to use.",
				PossibleValues: []any{"decimal", "currency", "percent", "unit"},
				Default:        "decimal",
			},
			{
				Name: "unit",
				Description: `The unit to use in unit formatting.
Possible values are core unit identifiers, defined in UTS #35, Part 2, Section 6.
A subset of units from the full list was selected for use in ECMAScript.
Pairs of simple units can be concatenated with "-per-" to make a compound unit.
There is no default value; if the style is "unit", the unit property must be provided.`,
				ValidateValue: isPositiveInteger,
			},
			{
				Name:           "unitDisplay",
				Description:    "The unit formatting style to use in unit formatting.",
				PossibleValues: []any{"long", "short", "narrow"},
				Default:        "short",
			},
			{
				Name: "minimumIntegerDigits",
				Description: `The minimum number of integer digits to use.
A value with a smaller number of integer digits than this number will be
left-padded with zeros (to the specified length) when formatted.`,
				ValidateValue: isPositiveInteger,
				Default:       1,
			},
			{
				Name: "minimumFractionDigits",
				Description: `The minimum number of fraction digits to use.
The default for plain number and percent formatting is 0;
the default for currency formatting is the number of minor unit digits provided by
the ISO 4217 currency code list (2 if the list doesn't provide that information).`,
				ValidateValue: isPositiveInteger,
			},
			{
				Name: "maximumFractionDigits",
				Description: `The maximum number of fraction digits to use.
The default for plain number formatting is the larger of minimumFractionDigits and 3;
the default for currency formatting is the larger of minimumFractionDigits and the number of
minor
unit digits provided by the ISO 4217 currency code list (2 if the list doesn't provide that
information);
the default for percent formatting is the larger of minimumFractionDigits and 0.`,
				ValidateValue: isPositiveInteger,
			},
			{
				Name:          "minimumSignificantDigits",
				Description:   `The minimum number of significant digits to use.`,
				ValidateValue: isPositiveInteger,
				Default:       1,
			},
			{
				Name:          "maximumSignificantDigits",
				Description:   `The maximum number of significant digits to use.`,
				ValidateValue: isPositiveInteger,
				Default:       21, //nolint:gomnd
			},
		},
	},
}

// TODO: supports only style and signDisplay options.
func numberF(input any, options map[string]any) (any, error) {
	num, err := castAs[float64](input)
	if err != nil {
		return nil, fmt.Errorf("convert input to float64: %w", err)
	}

	if len(options) == 0 {
		return num, nil
	}

	for optName := range options {
		switch optName {
		case "compactDisplay", "currency", "currencyDisplay", "currencySign", "notation", "numberingSystem",
			"unit", "unitDisplay", "minimumIntegerDigits", "minimumFractionDigits",
			"maximumFractionDigits", "minimumSignificantDigits", "maximumSignificantDigits":
			return nil, fmt.Errorf("option '%s' is not implemented", optName)
		}
	}

	var result string

	style, ok := options["style"]
	if !ok {
		style = "decimal"
	}

	switch style {
	case "decimal":
		result = strconv.FormatFloat(num, 'f', -1, 64)
	case "percent":
		result = fmt.Sprintf("%.2f%%", num*100) //nolint:gomnd
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
		return fmt.Errorf("value must be at least 0")
	}

	return nil
}
