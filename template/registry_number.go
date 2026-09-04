package template

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"strconv"
	"strings"

	"go.expect.digital/mf2"
	"golang.org/x/text/currency"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// parseNumberOperand parses resolved operand value.
func parseNumberOperand(operand *ResolvedValue) (float64, error) {
	errorf := func(format string, args ...any) (float64, error) {
		return 0, fmt.Errorf(format+": %w", append(args, mf2.ErrBadOperand)...)
	}

	var (
		number float64
		err    error
	)

	value := operand.value

	switch v := value.(type) {
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
	Select    string
	SelectErr error

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

	DisableSelect bool

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

func parseSelectOption(opts Options, options *numberOptions) error {
	if opts == nil || opts["select"] == nil {
		options.Select = "plural"

		return nil
	}

	if !opts.isLiteral("select") {
		options.SelectErr = fmt.Errorf("%w: select option must be a literal", mf2.ErrBadOption)
		options.DisableSelect = true
		options.Select = "plural"

		return nil
	}

	validate := oneOf("plural", "ordinal", "exact")

	var err error

	options.Select, err = opts.GetString("select", "plural", validate)
	if err != nil {
		return fmt.Errorf("%w: %w", mf2.ErrBadOption, err)
	}

	return nil
}

func parseNumberOptions(opts Options) (*numberOptions, error) {
	errorf := func(format string, args ...any) (*numberOptions, error) {
		return nil, fmt.Errorf("%w: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	validate := oneOf(
		"compactDisplay", "currency", "currencyDisplay", "currencySign", "notation", "numberingSystem",
		"signDisplay", "style", "unit", "unitDisplay", "minimumIntegerDigits", "minimumFractionDigits",
		"maximumFractionDigits", "minimumSignificantDigits", "maximumSignificantDigits", "select", "useGrouping",
	)

	for k := range opts {
		err := validate(k)
		if err != nil {
			return errorf("%w", err)
		}
	}

	var (
		err     error
		options numberOptions
	)

	err = parseSelectOption(opts, &options)
	if err != nil {
		return nil, err
	}

	useGroupings := oneOf("auto", "always", "never", "min2")

	options.UseGrouping, err = opts.GetString("useGrouping", "auto", useGroupings)
	if err != nil {
		return errorf("%w", err)
	}

	compactDisplays := oneOf("short", "long")

	options.CompactDisplay, err = opts.GetString("compactDisplay", "short", compactDisplays)
	if err != nil {
		return errorf("%w", err)
	}

	if curr, ok := opts["currency"]; ok {
		switch v := curr.value.(type) {
		default:
			return errorf("invalid currency type: %T", v)
		case string:
			options.Currency, err = currency.ParseISO(v)
			if err != nil {
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

	options.CurrencyDisplay, err = opts.GetString("currencyDisplay", "", currencyDisplays)
	if err != nil {
		return errorf("%w", err)
	}

	currencySigns := oneOf("standard", "accounting")

	options.CurrencySign, err = opts.GetString("currencySign", "standard", currencySigns)
	if err != nil {
		return errorf("%w", err)
	}

	notations := oneOf("standard", "scientific", "engineering", "compact")

	options.Notation, err = opts.GetString("notation", "standard", notations)
	if err != nil {
		return errorf("%w", err)
	}

	numberingSystems := oneOf(
		"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru", "hanidec", "khmr",
		"knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec", "telu", "thai", "tibt",
	)

	options.NumberingSystem, err = opts.GetString("numberingSystem", "", numberingSystems)
	if err != nil {
		return errorf("%w", err)
	}

	signDisplays := oneOf("auto", "always", "exceptZero", "negative", "never")

	options.SignDisplay, err = opts.GetString("signDisplay", "auto", signDisplays)
	if err != nil {
		return errorf("%w", err)
	}

	styles := oneOf("decimal", "percent")

	options.Style, err = opts.GetString("style", "decimal", styles)
	if err != nil {
		return errorf("%w", err)
	}

	options.Unit, err = opts.GetInt("unit", 0)
	if err != nil {
		return errorf("%w", err)
	}

	unitDisplays := oneOf("short", "narrow", "long")

	options.UnitDisplay, err = opts.GetString("unitDisplay", "short", unitDisplays)
	if err != nil {
		return errorf("%w", err)
	}

	err = parseDigitOptions(opts, &options)
	if err != nil {
		return errorf("%w", err)
	}

	return &options, nil
}

func parseDigitOptions(opts Options, options *numberOptions) error {
	var err error

	options.MinimumIntegerDigits, err = opts.GetInt("minimumIntegerDigits", 1, eqOrGreaterThan(1))
	if err != nil {
		return err
	}

	options.MinimumFractionDigits, err = opts.GetInt("minimumFractionDigits", 0, eqOrGreaterThan(0))
	if err != nil {
		return err
	}

	options.MinimumSignificantDigits, err = opts.GetInt("minimumSignificantDigits", 0, eqOrGreaterThan(1))
	if err != nil {
		return err
	}

	options.MaximumSignificantDigits, err = opts.GetInt("maximumSignificantDigits", -1, eqOrGreaterThan(1))
	if err != nil {
		return err
	}

	if options.MaximumSignificantDigits > 0 && options.MinimumSignificantDigits > options.MaximumSignificantDigits {
		return fmt.Errorf("minimumSignificantDigits (%d) cannot be greater than maximumSignificantDigits (%d)",
			options.MinimumSignificantDigits, options.MaximumSignificantDigits)
	}

	var maxFractionDigits int

	switch {
	case options.MinimumSignificantDigits > 0 || options.MaximumSignificantDigits != -1:
		maxFractionDigits = -1
	case options.Style == "decimal":
		maxFractionDigits = 3
	default:
		maxFractionDigits = 0
	}

	options.MaximumFractionDigits, err = opts.GetInt("maximumFractionDigits", maxFractionDigits, eqOrGreaterThan(0))
	if err != nil {
		return err
	}

	return nil
}

// minFractionDigitsForSigDigits returns the minimum fraction digits required to satisfy minSig significant digits,
// accounting for rounding when maxSig > 0.
func minFractionDigitsForSigDigits(val float64, minSig, maxSig int) int {
	if minSig <= 0 {
		return 0
	}

	val = math.Abs(val)
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return 0
	}

	if val == 0 {
		return max(0, minSig-1)
	}

	prec := -1
	if maxSig > 0 {
		prec = maxSig - 1
	}

	s := strconv.FormatFloat(val, 'e', prec, 64)

	_, expStr, ok := strings.Cut(s, "e")
	if !ok {
		return 0
	}

	exp, _ := strconv.Atoi(expStr)

	return max(0, minSig-(exp+1))
}

func applySignDisplay(result string, signDisplay string, value float64) string {
	switch signDisplay {
	case "auto", "negative":
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

// numberFunc is the implementation of the number function. Locale-sensitive number formatting.
func numberFunc(operand *ResolvedValue, options Options, locale language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec number function: "+format, args...)
	}

	value, err := parseNumberOperand(operand)
	if err != nil {
		return errorf("%w", err)
	}

	// Merge options from operand if operand was produced by :number or :integer
	var selectInheritedFromOperand bool

	if operand != nil && (operand.function == ":number" || operand.function == ":integer") && operand.options != nil {
		merged := maps.Clone(operand.options)
		if _, ok := merged["select"]; ok && (options == nil || options["select"] == nil) {
			selectInheritedFromOperand = true
		}

		maps.Copy(merged, options)
		options = merged
	}

	opts, err := parseNumberOptions(options)
	if err != nil {
		return errorf("%w", err)
	}

	if selectInheritedFromOperand {
		opts.SelectErr = fmt.Errorf("%w: select option cannot be inherited from operand", mf2.ErrBadOption)
		opts.DisableSelect = true
	}

	calcVal := value

	if opts.Style == "percent" {
		const percentMultiplier = 100

		calcVal = value * percentMultiplier
	}

	minFrac := max(opts.MinimumFractionDigits,
		minFractionDigitsForSigDigits(calcVal, opts.MinimumSignificantDigits, opts.MaximumSignificantDigits))

	maxFrac := opts.MaximumFractionDigits
	if maxFrac >= 0 && minFrac > maxFrac {
		maxFrac = minFrac
	}

	p := message.NewPrinter(locale)
	numberOpts := []number.Option{
		number.MinFractionDigits(minFrac),
		number.MaxFractionDigits(maxFrac),
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
		return applySignDisplay(p.Sprint(num), opts.SignDisplay, value)
	}

	selectKey := func(keys []string) string {
		if hasExactKey(keys) {
			return format()
		}

		if opts.Select == "exact" {
			return ""
		}

		scale := -1
		if opts.MaximumFractionDigits == 0 && minFrac == 0 {
			// most likely integer formatting
			scale = 0
		}

		digits := num.Digits(nil, locale, scale)

		fracDigits := max(int(digits.End-digits.Exp), minFrac)

		var form plural.Form

		if opts.Select == "ordinal" {
			form = plural.Ordinal.MatchDigits(locale, digits.Digits, int(digits.Exp), fracDigits)
		} else {
			form = plural.Cardinal.MatchDigits(locale, digits.Digits, int(digits.Exp), fracDigits)
		}

		return pluralFormString(form)
	}

	withFunc := withFunction(":number", options)

	var resOpts []ResolvedValueOpt

	resOpts = append(resOpts, WithFormat(format), WithValueDirection(DirLTR), withFunc)
	if !opts.DisableSelect {
		resOpts = append(resOpts, WithSelectKey(selectKey))
	}

	result := NewResolvedValue(value, resOpts...)
	if opts.SelectErr != nil {
		result.err = opts.SelectErr

		return result, opts.SelectErr
	}

	return result, nil
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
