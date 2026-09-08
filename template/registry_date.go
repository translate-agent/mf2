package template

import (
	"cmp"
	"fmt"
	"time"

	"go.expect.digital/intl"
	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

var (
	validDateOption = oneOf("style", "length", "timeZone", "calendar", "fields")
	validDateFields = oneOf(
		"weekday",
		"day-weekday",
		"month-day",
		"month-day-weekday",
		"year-month-day",
		"year-month-day-weekday",
	)
	validDateStyle = oneOf("full", "long", "medium", "short")
)

type dateOptions struct {
	// (default is UTC)
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	TimeZone *time.Location
	// The predefined date formatting style to use (full, long, medium, short).
	Style string
	// The fields to display (weekday, day-weekday, month-day, month-day-weekday, year-month-day, year-month-day-weekday).
	Fields string
}

// parseDateOptions parses :date options.
func parseDateOptions(options Options) (*dateOptions, error) {
	errorf := func(format string, args ...any) (*dateOptions, error) {
		return nil, fmt.Errorf("%w: parse options: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	for k := range options {
		err := validDateOption(k)
		if err != nil {
			return errorf("%w", err)
		}

		switch k {
		case "calendar":
			return errorf(`option "%s" is not implemented`, k)
		}
	}

	var (
		opts dateOptions
		err  error
	)

	if _, ok := options["fields"]; ok {
		if !options.isLiteral("fields") {
			return errorf(`option "fields" value must be a literal`)
		}
	}

	opts.Fields, err = options.GetString("fields", "year-month-day", validDateFields)
	if err != nil {
		return errorf("%w", err)
	}

	if opts.Fields == "month-day-weekday" || opts.Fields == "year-month-day-weekday" {
		return errorf(`option "fields" with value "%s" is not implemented`, opts.Fields)
	}

	var style, length string

	if _, ok := options["style"]; ok {
		if !options.isLiteral("style") {
			return errorf(`option "style" value must be a literal`)
		}

		style, err = options.GetString("style", "", validDateStyle)
		if err != nil {
			return errorf("%w", err)
		}
	}

	if _, ok := options["length"]; ok {
		if !options.isLiteral("length") {
			return errorf(`option "length" value must be a literal`)
		}

		length, err = options.GetString("length", "", validDateStyle)
		if err != nil {
			return errorf("%w", err)
		}
	}

	opts.Style = cmp.Or(style, length, "medium")

	opts.TimeZone, err = getTZ(options)
	if err != nil {
		return errorf("%w", err)
	}

	return &opts, nil
}

// dateFunc is the implementation of the date function. Locale-sensitive date formatting.
func dateFunc(operand *ResolvedValue, options Options, locale language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec date function: "+format, args...)
	}

	// NOTE(mvilks): operand parsing is the same as for datetime registry function
	value, err := parseDatetimeOperand(operand)
	if err != nil {
		return errorf("%w", err)
	}

	opts, err := parseDateOptions(options)
	if err != nil {
		return errorf("%w", err)
	}

	value = value.In(opts.TimeZone)

	var intlOpts intl.Options

	switch opts.Fields {
	case "weekday":
		switch opts.Style {
		case "full", "long":
			intlOpts.Weekday = intl.WeekdayLong
		case "medium", "short":
			intlOpts.Weekday = intl.WeekdayShort
		}
	case "day-weekday":
		intlOpts.Day = intl.DayNumeric
		switch opts.Style {
		case "full", "long":
			intlOpts.Weekday = intl.WeekdayLong
		case "medium", "short":
			intlOpts.Weekday = intl.WeekdayShort
		}
	case "month-day":
		intlOpts.Day = intl.DayNumeric
		switch opts.Style {
		case "full", "long":
			intlOpts.Month = intl.MonthLong
		case "medium":
			intlOpts.Month = intl.MonthShort
		case "short":
			intlOpts.Month = intl.MonthNumeric
		}
	case "year-month-day":
		intlOpts.Day = intl.DayNumeric
		switch opts.Style {
		case "full", "long":
			intlOpts.Year = intl.YearNumeric
			intlOpts.Month = intl.MonthLong
		case "medium":
			intlOpts.Year = intl.YearNumeric
			intlOpts.Month = intl.MonthShort
		case "short":
			intlOpts.Year = intl.Year2Digit
			intlOpts.Month = intl.MonthNumeric
		}
	}

	formatter := intl.NewDateTimeFormat(locale, intlOpts)

	format := func() string {
		return formatter.Format(value)
	}

	return NewResolvedValue(value, WithFormat(format)), nil
}
