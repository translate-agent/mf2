package template

import (
	"encoding/json"
	"fmt"

	"go.expect.digital/mf2"
	"golang.org/x/text/currency"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// parseNumberOperand parses resolved operand value.
func parseNumberOperand(operand any) (float64, error) {
	errorf := func(format string, args ...any) (float64, error) {
		return 0, fmt.Errorf(format+": %w", append(args, mf2.ErrBadOperand)...)
	}

	var (
		number float64
		err    error
	)

	switch v := operand.(type) {
	default:
		number, err = castAs[float64](v)
		if err != nil {
			return errorf("unsupported operand type %T: %w", v, err)
		}
	case nil:
		return errorf("operand is required")
	case string:
		err = json.Unmarshal([]byte(v), &number)
		if err != nil {
			return errorf(`parse number "%s": %w`, operand, err)
		}
	}

	return number, nil
}

type numberOptions struct {
	// Only used when notation is "compact" (short, long).
	CompactDisplay string
	// How to display the currency in currency formatting.
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	CurrencyDisplay string
	// In many locales, accounting format means to wrap the number with parentheses
	// instead of appending a minus sign. You can enable this formatting by setting the
	// currencySign option to "accounting".
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	CurrencySign string
	// The formatting that should be displayed for the number (standard, scientific, engineering, compact).
	Notation string
	// Numbering system to use.
	NumberingSystem string
	// When to display the sign for the number. "negative" value is Experimental.
	// (auto, always, exceptZero, negative, never)
	SignDisplay string
	// The formatting style to use.
	Style string
	// The unit formatting style to use in unit formatting (decimal, percent).
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	UnitDisplay string
	// (plural, ordinal, exact)
	Select string
	// (auto, always, never, min2)
	UseGrouping string
	// The currency to use in currency formatting.
	// Possible values are the ISO 4217 currency codes, such as "USD" for the US dollar,
	// "EUR" for the euro, or "CNY" for the Chinese RMB — see the
	// Current currency &amp; funds code list
	// (https://www.unicode.org/cldr/charts/latest/supplemental/detailed_territory_currency_information.html).
	// There is no default value; if the style is "currency", the currency property must be provided.
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	Currency currency.Unit
	// The unit to use in unit formatting.
	// Possible values are core unit identifiers, defined in UTS #35, Part 2, Section 6.
	// A subset of units from the full list was selected for use in ECMAScript.
	// Pairs of simple units can be concatenated with "-per-" to make a compound unit.
	// There is no default value; if the style is "unit", the unit property must be provided.
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	Unit int
	// The minimum number of integer digits to use.
	// A value with a smaller number of integer digits than this number will be
	// left-padded with zeros (to the specified length) when formatted.
	MinimumIntegerDigits int
	// The minimum number of fraction digits to use.
	// The default for plain number and percent formatting is 0;
	// the default for currency formatting is the number of minor unit digits provided by
	// the ISO 4217 currency code list (2 if the list doesn't provide that information).
	MinimumFractionDigits int
	// The maximum number of fraction digits to use.
	// The default for plain number formatting is the larger of minimumFractionDigits and 3;
	// the default for currency formatting is the larger of minimumFractionDigits and the number of
	// minor
	// unit digits provided by the ISO 4217 currency code list (2 if the list doesn't provide that
	// information);
	// the default for percent formatting is the larger of minimumFractionDigits and 0.
	MaximumFractionDigits int
	// The minimum number of significant digits to use.
	MinimumSignificantDigits int
	// The maximum number of significant digits to use.
	MaximumSignificantDigits int
}

func parseNumberOptions(opts Options) (*numberOptions, error) {
	errorf := func(format string, args ...any) (*numberOptions, error) {
		return nil, fmt.Errorf("%w: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	for k := range opts {
		switch k {
		default:
			return errorf("unsupported option: %s", k)
		case "compactDisplay", "currency", "currencyDisplay", "currencySign", "notation", "numberingSystem",
			"signDisplay", "style", "unit", "unitDisplay", "minimumIntegerDigits", "minimumFractionDigits",
			"maximumFractionDigits", "minimumSignificantDigits", "maximumSignificantDigits", "select", "useGrouping": // noop
		}
	}

	var (
		err     error
		options numberOptions
	)

	selects := oneOf("plural", "ordinal", "exact")
	if options.Select, err = opts.GetString("select", "plural", selects); err != nil {
		return errorf("%w", err)
	}

	useGroupings := oneOf("auto", "always", "never", "min2")
	if options.UseGrouping, err = opts.GetString("useGrouping", "auto", useGroupings); err != nil {
		return errorf("%w", err)
	}

	compactDisplays := oneOf("short", "long")
	if options.CompactDisplay, err = opts.GetString("compactDisplay", "short", compactDisplays); err != nil {
		return errorf("%w", err)
	}

	if curr, ok := opts["currency"]; ok {
		switch v := curr.(type) {
		default:
			return errorf("invalid currency type: %T", v)
		case string:
			if options.Currency, err = currency.ParseISO(v); err != nil {
				return errorf("invalid currency value: %s", v)
			}

			if options.Currency == currency.XXX {
				return errorf("empty currency value")
			}
		case currency.Unit:
			options.Currency = v
		}
	}

	currencyDisplays := oneOf("code", "symbol", "narrowSymbol", "name")
	if options.CurrencyDisplay, err = opts.GetString("currencyDisplay", "", currencyDisplays); err != nil {
		return errorf("%w", err)
	}

	currencySigns := oneOf("standard", "accounting")
	if options.CurrencySign, err = opts.GetString("currencySign", "standard", currencySigns); err != nil {
		return errorf("%w", err)
	}

	notations := oneOf("standard", "scientific", "engineering", "compact")
	if options.Notation, err = opts.GetString("notation", "standard", notations); err != nil {
		return errorf("%w", err)
	}

	numberingSystems := oneOf(
		"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru", "hanidec", "khmr",
		"knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec", "telu", "thai", "tibt",
	)
	if options.NumberingSystem, err = opts.GetString("numberingSystem", "", numberingSystems); err != nil {
		return errorf("%w", err)
	}

	signDisplays := oneOf("auto", "always", "exceptZero", "negative", "never")
	if options.SignDisplay, err = opts.GetString("signDisplay", "auto", signDisplays); err != nil {
		return errorf("%w", err)
	}

	styles := oneOf("decimal", "percent")
	if options.Style, err = opts.GetString("style", "decimal", styles); err != nil {
		return errorf("%w", err)
	}

	if options.Unit, err = opts.GetInt("unit", 0); err != nil {
		return errorf("%w", err)
	}

	unitDisplays := oneOf("short", "narrow")
	if options.UnitDisplay, err = opts.GetString("unitDisplay", "short", unitDisplays); err != nil {
		return errorf("%w", err)
	}

	if options.MinimumIntegerDigits, err = opts.GetInt("minimumIntegerDigits", 1, eqOrGreaterThan(1)); err != nil {
		return errorf("%w", err)
	}

	if options.MinimumFractionDigits, err = opts.GetInt("minimumFractionDigits", 0, eqOrGreaterThan(0)); err != nil {
		return errorf("%w", err)
	}

	var maxFractionDigits int // percent default

	if options.Style == "decimal" {
		maxFractionDigits = 3 // decimal default
	}

	options.MaximumFractionDigits, err = opts.GetInt("maximumFractionDigits", maxFractionDigits, eqOrGreaterThan(0))
	if err != nil {
		return errorf("%w", err)
	}

	if options.MinimumSignificantDigits, err = opts.GetInt("minimumSignificantDigits", 1, eqOrGreaterThan(1)); err != nil {
		return errorf("%w", err)
	}

	options.MaximumSignificantDigits, err = opts.GetInt("maximumSignificantDigits", -1)
	if err != nil {
		return errorf("%w", err)
	}

	return &options, nil
}

// numberFunc is the implementation of the number function. Locale-sensitive number formatting.
func numberFunc(operand *ResolvedValue, options Options, locale language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec number function: "+format, args...)
	}

	value, err := parseNumberOperand(operand.value)
	if err != nil {
		return errorf("%w", err)
	}

	opts, err := parseNumberOptions(options)
	if err != nil {
		return errorf("%w", err)
	}

	p := message.NewPrinter(locale)
	numberOpts := []number.Option{
		number.MinFractionDigits(opts.MinimumFractionDigits),
		number.MaxFractionDigits(opts.MaximumFractionDigits),
		number.MinIntegerDigits(opts.MinimumIntegerDigits),
		number.Precision(opts.MaximumSignificantDigits),
	}

	var num number.Formatter

	switch opts.Style {
	default:
		return errorf(`option style "%s" is not implemented`, opts.Style)
	case "decimal":
		num = number.Decimal(value, numberOpts...)
	case "percent":
		num = number.Percent(value, numberOpts...)
	}

	format := func() string {
		result := p.Sprint(num)

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

		return result
	}

	selectKey := func(keys []string) string {
		if hasExactKey(keys) {
			return format()
		}

		scale := -1
		if opts.MaximumFractionDigits == 0 {
			// most likely integer formatting
			scale = 0
		}

		digits := num.Digits(nil, locale, scale)
		form := plural.Cardinal.MatchDigits(locale, digits.Digits, int(digits.Exp), int(digits.End-digits.Exp))

		return pluralFormString(form)
	}

	return NewResolvedValue(value, WithFormat(format), WithSelectKey(selectKey)), nil
}

// hasExactKey returns true if the variant keys contain exact value besides the plural categories.
func hasExactKey(keys []string) bool {
	for _, key := range keys {
		switch key {
		default:
			return true
		case "zero", "one", "two", "few", "many", "other": // check next key
		}
	}

	return false
}
