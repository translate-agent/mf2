package template

import (
	"fmt"
	"time"

	"golang.org/x/text/language"
)

// dateFunc is the implementation of the date function. Locale-sensitive date formatting.
var dateRegistryFunc = RegistryFunc{
	Format: dateFunc,
}

type dateOptions struct {
	// (default is system default time zone or UTC)
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

	opts.TimeZone = time.UTC

	if v, ok := options["timeZone"]; ok {
		switch tz := v.(type) {
		default:
			return errorf("want timeZone as string or *time.Location, got %T", v)
		case *time.Location:
			opts.TimeZone = tz
		case string:
			if opts.TimeZone, err = time.LoadLocation(tz); err != nil {
				return errorf("load TZ data for %s: %w", tz, err)
			}
		}
	}

	return &opts, nil
}

func dateFunc(operand any, options Options, locale language.Tag) (any, error) {
	errorf := func(format string, args ...any) (any, error) {
		return "", fmt.Errorf("exec date function: "+format, args...)
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

	if opts == nil {
		opts = &dateOptions{Style: "short"}
	}

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

	return value.Format(layout), nil
}
