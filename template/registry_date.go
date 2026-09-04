package template

import (
	"fmt"
	"time"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
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
}

// parseDateOptions parses :date options.
func parseDateOptions(options Options) (*dateOptions, error) {
	errorf := func(format string, args ...any) (*dateOptions, error) {
		return nil, fmt.Errorf("%w: parse options: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	validate := oneOf("style", "length", "timeZone", "calendar", "fields")

	for k := range options {
		err := validate(k)
		if err != nil {
			return errorf("%w", err)
		}

		switch k {
		case "calendar", "fields":
			return errorf(`option "%s" is not implemented`, k)
		}
	}

	var (
		opts dateOptions
		err  error
	)

	styles := oneOf("full", "long", "medium", "short")

	if _, ok := options["length"]; ok && options["style"] == nil {
		opts.Style, err = options.GetString("length", "short", styles)
	} else {
		opts.Style, err = options.GetString("style", "short", styles)
	}

	if err != nil {
		return errorf("%w", err)
	}

	opts.TimeZone, err = getTZ(options)
	if err != nil {
		return errorf("%w", err)
	}

	return &opts, nil
}

// dateFunc is the implementation of the date function. Locale-sensitive date formatting.
func dateFunc(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
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

	format := func() string {
		var layout string

		switch opts.Style {
		case "full":
			layout = "Monday, 02 January 2006"
		case "long":
			layout = "02 January 2006"
		case "medium":
			layout = "02 Jan 2006"
		case "short":
			layout = "02/01/06"
		}

		value = value.In(opts.TimeZone)

		return value.Format(layout)
	}

	return NewResolvedValue(value, WithFormat(format)), nil
}
