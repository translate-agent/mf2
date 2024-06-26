package template

import (
	"fmt"
	"time"

	"golang.org/x/text/language"
)

// See ".message-format-wg/spec/registry.xml".

// datetimeFunc is the implementation of the datetime function. Locale-sensitive date and time formatting.
var datetimeRegistryFunc = RegistryFunc{
	Format: datetimeFunc,
}

type datetimeOptions struct {
	// (default is system default time zone or UTC)
	//
	// NOTE: The option is not part of the default registry.
	// Implementations SHOULD avoid creating options that conflict with these, but
	// are encouraged to track development of these options during Tech Preview.
	TimeZone *time.Location
	// The predefined date formatting style to use (full, long, medium, short).
	DateStyle string
	// The predefined time formatting style to use (full, long, medium, short).
	TimeStyle string
	// The hour cycle to use (h11, h12, h23, h24).
	HourCycle string
	// DayPeriod is mentioned in registry.xml, but NOT in registry.md.
	// See https://github.com/unicode-org/message-format-wg/issues/596
	//
	// The formatting style used for day periods like "in the morning", "am", "noon", "n" etc.
	DayPeriod string
	// The representation of the weekday (long, short, narrow).
	Weekday string
	// The representation of the era (long, short, narrow).
	Era string
	// The representation of the year (numeric, 2-digit).
	Year string
	// The representation of the month (numeric, 2-digit).
	Month string
	// The representation of the day (numeric, 2-digit, long, short, narrow).
	Day string
	// The representation of the hour (numeric, 2-digit).
	Hour string
	// The representation of the minute (numeric, 2-digit).
	Minute string
	// The representation of the second (numeric, 2-digit).
	Second string
	// The localized representation of the time zone name
	// (long, short, shortOffset, longOffset, shortGeneric, longGeneric).
	TimeZoneName string
	// The number of fractional seconds to display (1, 2, 3).
	FractionalSecondDigits int
}

func parseDatetimeInput(input any) (time.Time, error) {
	if input == nil {
		return time.Time{}, fmt.Errorf("input is required: %w", ErrOperandMismatch)
	}

	switch v := input.(type) {
	default:
		return time.Time{}, fmt.Errorf("unsupported datetime type %T: %w", input, ErrOperandMismatch)
	case string:
		// layout is quick and dirty, does not conform with ISO 8601 fully as required
		t, err := time.Parse(time.RFC3339[:len(v)], v)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse datetime %s: %w", v, ErrOperandMismatch)
		}

		return t, nil
	case time.Time:
		return v, nil
	}
}

// parseDatetimeOptions parses :datetime options.
func parseDatetimeOptions(options Options) (*datetimeOptions, error) {
	for opt := range options {
		switch opt {
		case "calendar", "numberingSystem", "hourCycle", "dayPeriod", "weekday", "era",
			"year", "month", "day", "hour", "minute", "second", "fractionalSecondDigits":
			return nil, fmt.Errorf("option '%s' is not implemented", opt)
		}
	}

	var (
		opts datetimeOptions
		err  error
	)

	dateStyles := oneOf("full", "long", "medium", "short")
	if opts.DateStyle, err = options.GetString("dateStyle", "", dateStyles); err != nil {
		return nil, err
	}

	timeStyles := oneOf("full", "long", "medium", "short")
	if opts.TimeStyle, err = options.GetString("timeStyle", "", timeStyles); err != nil {
		return nil, err
	}

	opts.TimeZone = time.UTC

	if v, ok := options["timeZone"]; ok {
		switch tz := v.(type) {
		default:
			return nil, fmt.Errorf("unsupported timeZone type: %T", v)
		case *time.Location:
			opts.TimeZone = tz
		case string:
			if opts.TimeZone, err = time.LoadLocation(tz); err != nil {
				return nil, fmt.Errorf("load tz %s: %w", tz, err)
			}
		}
	}

	hourCycles := oneOf("h11", "h12", "h23", "h24")
	if opts.HourCycle, err = options.GetString("hourCycle", "", hourCycles); err != nil {
		return nil, err
	}

	dayPeriods := oneOf("short", "long")
	if opts.DayPeriod, err = options.GetString("dayPeriod", "", dayPeriods); err != nil {
		return nil, err
	}

	weekdays := oneOf("narrow", "short", "long")
	if opts.Weekday, err = options.GetString("weekday", "", weekdays); err != nil {
		return nil, err
	}

	eras := oneOf("narrow", "short", "long")
	if opts.Era, err = options.GetString("era", "", eras); err != nil {
		return nil, err
	}

	years := oneOf("numeric", "2-digit")
	if opts.Year, err = options.GetString("year", "", years); err != nil {
		return nil, err
	}

	months := oneOf("numeric", "2-digit", "narrow", "short", "long")
	if opts.Month, err = options.GetString("month", "", months); err != nil {
		return nil, err
	}

	days := oneOf("numeric", "2-digit")
	if opts.Day, err = options.GetString("day", "", days); err != nil {
		return nil, err
	}

	hours := oneOf("numeric", "2-digit")
	if opts.Hour, err = options.GetString("hour", "", hours); err != nil {
		return nil, err
	}

	minutes := oneOf("numeric", "2-digit")
	if opts.Minute, err = options.GetString("minute", "", minutes); err != nil {
		return nil, err
	}

	seconds := oneOf("numeric", "2-digit")
	if opts.Second, err = options.GetString("second", "", seconds); err != nil {
		return nil, err
	}

	//nolint:mnd
	if opts.FractionalSecondDigits, err = options.GetInt("fractionalSecondDigits", 0, oneOf(1, 2, 3)); err != nil {
		return nil, err
	}

	timeZoneNames := oneOf("long", "short", "shortOffset", "longOffset", "shortGeneric", "longGeneric")
	if opts.TimeZoneName, err = options.GetString("timeZoneName", "", timeZoneNames); err != nil {
		return nil, err
	}

	return &opts, nil
}

func datetimeFunc(input any, options Options, locale language.Tag) (any, error) {
	value, err := parseDatetimeInput(input)
	if err != nil {
		return "", err
	}

	if len(options) == 0 {
		return fmt.Sprint(input), nil
	}

	opts, err := parseDatetimeOptions(options)
	if err != nil {
		return "", err
	}

	var layout string

	switch opts.DateStyle {
	case "full":
		layout = "Monday, 02 January 2006"
	case "long":
		layout = "02 January 2006"
	case "medium":
		layout = "02 Jan 2006"
	case "short":
		layout = "02/01/06"
	}

	if len(opts.TimeStyle) > 0 {
		if len(layout) > 0 {
			layout += " "
		}

		switch opts.TimeStyle {
		case "full", "long":
			layout += "15:04:05"
		case "medium":
			layout += "15:04"
		case "short":
			layout += "15"
		}
	}

	value = value.In(opts.TimeZone)

	return value.Format(layout), nil
}
