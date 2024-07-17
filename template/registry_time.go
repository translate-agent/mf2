package template

import (
	"fmt"
	"time"

	"golang.org/x/text/language"
)

// timeFunc is the implementation of the time function. Locale-sensitive time formatting.
var timeRegistryFunc = RegistryFunc{
	Format: timeFunc,
}

type timeOptions struct {
	// (default is UTC)
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	TimeZone *time.Location
	// The predefined time formatting style to use (full, long, medium, short).
	Style string
}

// parseTimeOptions parses :time options.
func parseTimeOptions(options Options) (*timeOptions, error) {
	errorf := func(format string, args ...any) (*timeOptions, error) {
		return nil, fmt.Errorf("parse options: "+format, args...)
	}

	var (
		opts timeOptions
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

func timeFunc(operand any, options Options, locale language.Tag) (any, error) {
	errorf := func(format string, args ...any) (any, error) {
		return "", fmt.Errorf("exec time function: "+format, args...)
	}

	// NOTE(mvilks): operand parsing is the same as for datetime registry function
	value, err := parseDatetimeOperand(operand)
	if err != nil {
		return errorf("%w", err)
	}

	opts, err := parseTimeOptions(options)
	if err != nil {
		return errorf("%w", err)
	}

	var layout string

	// time styles as per Intl.DateTimeFormat
	// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat
	switch opts.Style {
	case "full":
		layout = "15:04:05 MST"
	case "long":
		layout = "15:04:05 -0700"
	case "medium":
		layout = "15:04:05"
	case "short":
		layout = "15:04"
	}

	value = value.In(opts.TimeZone)

	return value.Format(layout), nil
}
