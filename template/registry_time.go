package template

import (
	"fmt"
	"time"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

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
		return nil, fmt.Errorf("%w: parse options: "+format, append([]any{mf2.ErrBadOption}, args...)...)
	}

	validate := oneOf("style", "precision", "timeZone", "calendar", "hour12", "timeZoneStyle")

	for k := range options {
		err := validate(k)
		if err != nil {
			return errorf("%w", err)
		}

		switch k {
		case "calendar", "hour12", "timeZoneStyle":
			return errorf(`option "%s" is not implemented`, k)
		}
	}

	var (
		opts      timeOptions
		precision string
		err       error
	)

	styles := oneOf("full", "long", "medium", "short")
	precisions := oneOf("hour", "minute", "second")

	if _, ok := options["precision"]; ok && options["style"] == nil {
		precision, err = options.GetString("precision", "minute", precisions)
		if err != nil {
			return errorf("%w", err)
		}

		switch precision {
		case "second":
			opts.Style = "medium"
		case "minute", "hour":
			opts.Style = "short"
		}
	} else {
		opts.Style, err = options.GetString("style", "short", styles)
		if err != nil {
			return errorf("%w", err)
		}
	}

	opts.TimeZone, err = getTZ(options)
	if err != nil {
		return errorf("%w", err)
	}

	return &opts, nil
}

// timeFunc is the implementation of the time function. Locale-sensitive time formatting.
func timeFunc(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec time function: "+format, args...)
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

	format := func() string {
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

		return value.Format(layout)
	}

	return NewResolvedValue(value, WithFormat(format)), nil
}
