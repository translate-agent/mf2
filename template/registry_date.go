package template

import (
	"fmt"
	"time"

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
		return nil, fmt.Errorf("parse options: "+format, args...)
	}

	var (
		opts dateOptions
		err  error
	)

	styles := oneOf("full", "long", "medium", "short")
	if opts.Style, err = options.GetString("style", "short", styles); err != nil {
		return errorf("%w", err)
	}

	if opts.TimeZone, err = getTZ(options); err != nil {
		return errorf("%w", err)
	}

	return &opts, nil
}

// dateFunc is the implementation of the date function. Locale-sensitive date formatting.
func dateFunc(operand any, options Options, _ language.Tag) (*ResolvedValue, error) {
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
