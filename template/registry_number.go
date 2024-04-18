package template

import (
	"errors"
	"fmt"
	"reflect"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// https://github.com/unicode-org/message-format-wg/blob/1dc84e648a6f98d74ac62306abaacc0bed8e4fc5/spec/registry.xml#L147

// numberRegistryFunc is the implementation of the number function. Locale-sensitive number formatting.
var numberRegistryFunc = RegistryFunc{
	Format: numberFunc,
}

func parseNumberInput(input any) (float64, error) {
	if input == nil {
		return 0, fmt.Errorf("input is required: %w", ErrOperandMismatch)
	}

	v, err := castAs[float64](input)
	if err != nil {
		return 0, fmt.Errorf("unsupported type %T: %w: %w", input, err, ErrOperandMismatch)
	}

	return v, nil
}

type numberOptions struct {
	CompactDisplay           string
	CurrencyDisplay          string
	CurrencySign             string
	Notation                 string
	NumberingSystem          string
	SignDisplay              string
	Style                    string
	UnitDisplay              string
	Currency                 currency.Unit
	Unit                     int
	MinimumIntegerDigits     int
	MinimumFractionDigits    int
	MaximumFractionDigits    int
	MinimumSignificantDigits int
	MaximumSignificantDigits int
}

func parseNumberOptions(opts Options) (*numberOptions, error) {
	for k := range opts {
		switch k {
		default:
			return nil, fmt.Errorf("unsupported option: %s", k)
		case "compactDisplay", "currency", "currencyDisplay", "currencySign", "notation", "numberingSystem",
			"signDisplay", "style", "unit", "unitDisplay", "minimumIntegerDigits", "minimumFractionDigits",
			"maximumFractionDigits", "minimumSignificantDigits", "maximumSignificantDigits": // noop
		}
	}

	var (
		err     error
		options numberOptions
	)

	// Only used when notation is "compact".
	compactDisplays := oneOf("short", "long")
	if options.CompactDisplay, err = opts.GetString("compactDisplay", "short", compactDisplays); err != nil {
		return nil, err
	}

	// The currency to use in currency formatting.
	// Possible values are the ISO 4217 currency codes, such as "USD" for the US dollar,
	// "EUR" for the euro, or "CNY" for the Chinese RMB — see the
	// Current currency &amp; funds code list
	// (https://www.unicode.org/cldr/charts/latest/supplemental/detailed_territory_currency_information.html).
	// There is no default value; if the style is "currency", the currency property must be provided.
	if curr, ok := opts["currency"]; ok {
		switch v := curr.(type) {
		default:
			return nil, fmt.Errorf("invalid currency type: %T", v)
		case string:
			if options.Currency, err = currency.ParseISO(v); err != nil {
				return nil, fmt.Errorf("invalid currency value: %s", v)
			}

			if options.Currency == currency.XXX {
				return nil, errors.New("empty currency value")
			}
		case currency.Unit:
			options.Currency = v
		}
	}

	// How to display the currency in currency formatting.
	currencyDisplays := oneOf("code", "symbol", "narrowSymbol", "name")
	if options.CurrencyDisplay, err = opts.GetString("currencyDisplay", "", currencyDisplays); err != nil {
		return nil, err
	}

	// In many locales, accounting format means to wrap the number with parentheses
	// instead of appending a minus sign. You can enable this formatting by setting the
	// currencySign option to "accounting".
	currencySigns := oneOf("standard", "accounting")
	if options.CurrencySign, err = opts.GetString("currencySign", "standard", currencySigns); err != nil {
		return nil, err
	}

	// The formatting that should be displayed for the number.
	notations := oneOf("standard", "scientific", "engineering", "compact")
	if options.Notation, err = opts.GetString("notation", "standard", notations); err != nil {
		return nil, err
	}

	// Numbering system to use.
	numberingSystems := oneOf(
		"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru", "hanidec", "khmr",
		"knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec", "telu", "thai", "tibt",
	)
	if options.NumberingSystem, err = opts.GetString("numberingSystem", "", numberingSystems); err != nil {
		return nil, err
	}

	// When to display the sign for the number. "negative" value is Experimental.
	signDisplays := oneOf("auto", "always", "exceptZero", "negative", "never")
	if options.SignDisplay, err = opts.GetString("signDisplay", "auto", signDisplays); err != nil {
		return nil, err
	}

	// The formatting style to use.
	styles := oneOf("decimal", "percent")
	if options.Style, err = opts.GetString("style", "decimal", styles); err != nil {
		return nil, err
	}

	// The unit to use in unit formatting.
	// Possible values are core unit identifiers, defined in UTS #35, Part 2, Section 6.
	// A subset of units from the full list was selected for use in ECMAScript.
	// Pairs of simple units can be concatenated with "-per-" to make a compound unit.
	// There is no default value; if the style is "unit", the unit property must be provided.
	if options.Unit, err = opts.GetInt("unit", 0); err != nil {
		return nil, err
	}

	// The unit formatting style to use in unit formatting.
	unitDisplays := oneOf("short", "narrow")
	if options.UnitDisplay, err = opts.GetString("unitDisplay", "short", unitDisplays); err != nil {
		return nil, err
	}

	// The minimum number of integer digits to use.
	// A value with a smaller number of integer digits than this number will be
	// left-padded with zeros (to the specified length) when formatted.
	if options.MinimumIntegerDigits, err = opts.GetInt("minimumIntegerDigits", 1, eqOrGreaterThan(1)); err != nil {
		return nil, err
	}

	// The minimum number of fraction digits to use.
	// The default for plain number and percent formatting is 0;
	// the default for currency formatting is the number of minor unit digits provided by
	// the ISO 4217 currency code list (2 if the list doesn't provide that information).
	if options.MinimumFractionDigits, err = opts.GetInt("minimumFractionDigits", 0, eqOrGreaterThan(0)); err != nil {
		return nil, err
	}

	// The maximum number of fraction digits to use.
	// The default for plain number formatting is the larger of minimumFractionDigits and 3;
	// the default for currency formatting is the larger of minimumFractionDigits and the number of
	// minor
	// unit digits provided by the ISO 4217 currency code list (2 if the list doesn't provide that
	// information);
	// the default for percent formatting is the larger of minimumFractionDigits and 0.
	var maxFractionDigits int // percent default

	if options.Style == "decimal" {
		maxFractionDigits = 3 // decimal default
	}

	options.MaximumFractionDigits, err = opts.GetInt("maximumFractionDigits", maxFractionDigits, eqOrGreaterThan(0))
	if err != nil {
		return nil, err
	}

	// The minimum number of significant digits to use.
	if options.MinimumSignificantDigits, err = opts.GetInt("minimumSignificantDigits", 1, eqOrGreaterThan(1)); err != nil {
		return nil, err
	}

	// The maximum number of significant digits to use.
	options.MaximumSignificantDigits, err = opts.GetInt("maximumSignificantDigits", -1)
	if err != nil {
		return nil, err
	}

	return &options, nil
}

func numberFunc(input any, options Options, locale language.Tag) (any, error) {
	value, err := parseNumberInput(input)
	if err != nil {
		return nil, err
	}

	opts, err := parseNumberOptions(options)
	if err != nil {
		return nil, err
	}

	var result string

	p := message.NewPrinter(locale)
	numberOpts := []number.Option{
		number.MinFractionDigits(opts.MinimumFractionDigits),
		number.MaxFractionDigits(opts.MaximumFractionDigits),
		number.MinIntegerDigits(opts.MinimumIntegerDigits),
		number.Precision(opts.MaximumSignificantDigits),
	}

	switch opts.Style {
	case "decimal":
		result = p.Sprint(number.Decimal(value, numberOpts...))
	case "percent":
		result = p.Sprint(number.Percent(value, numberOpts...))
	default:
		return nil, fmt.Errorf("style '%s' is not implemented", opts.Style)
	}

	switch opts.SignDisplay {
	case "auto":
	case "negative":
	case "always":
		if value >= 0 {
			result = "+" + result
		}
	case "exceptZero":
		if value > 0 {
			result = "+" + result
		}
	case "never":
		if value < 0 {
			result = result[1:]
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
		return zeroVal, fmt.Errorf("convert %v to %T", v.Type(), zeroVal)
	}

	v = v.Convert(typ)

	return v.Interface().(T), nil //nolint:forcetypeassert
}
