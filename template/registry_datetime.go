package template

import (
	"fmt"
	"time"

	"go.expect.digital/intl"
	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

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

// parseDatetimeOperand parses resolved operand value.
func parseDatetimeOperand(operand *ResolvedValue) (time.Time, error) {
	errorf := func(format string, args ...any) (time.Time, error) {
		return time.Time{}, fmt.Errorf(format+": %w", append(args, mf2.ErrBadOperand)...)
	}

	value := operand.value

	if value == nil {
		return errorf("operand is required")
	}

	switch v := value.(type) {
	default:
		return errorf("unsupported operand type %T", value)
	case string:
		// layout is quick and dirty, does not conform with ISO 8601 fully as required
		t, err := time.Parse(time.RFC3339[:len(v)], v)
		if err != nil {
			return errorf(`parse operand "%s"`, v)
		}

		return t, nil
	case time.Time:
		return v, nil
	}
}

func validateDatetimeOptions(options Options) error {
	validate := oneOf(
		"calendar", "numberingSystem", "hourCycle", "dayPeriod", "weekday", "era",
		"month", "hour", "minute", "second", "fractionalSecondDigits",
		"dateStyle", "timeStyle", "timeZone", "year", "day", "timeZoneName",
		"dateLength", "timePrecision", "dateFields", "hour12", "timeZoneStyle",
	)

	for opt := range options {
		err := validate(opt)
		if err != nil {
			return err
		}

		switch opt {
		case "calendar", "numberingSystem", "hourCycle", "dayPeriod", "weekday", "era",
			"month", "hour", "minute", "second", "fractionalSecondDigits",
			"dateFields", "hour12", "timeZoneStyle":
			return fmt.Errorf(`option "%s" is not implemented`, opt)
		}
	}

	return nil
}

// parseDatetimeOptions parses :datetime options.
func parseDatetimeOptions(options Options) (*datetimeOptions, error) {
	errorf := func(err error) (*datetimeOptions, error) {
		return nil, fmt.Errorf("%w: parse options: %w", mf2.ErrBadOption, err)
	}

	if len(options) == 0 {
		return &datetimeOptions{DateStyle: "medium", TimeStyle: "short"}, nil
	}

	err := validateDatetimeOptions(options)
	if err != nil {
		return errorf(err)
	}

	var (
		opts      datetimeOptions
		precision string
	)

	dateStyles := oneOf("full", "long", "medium", "short")

	if _, ok := options["dateLength"]; ok && options["dateStyle"] == nil {
		opts.DateStyle, err = options.GetString("dateLength", "", dateStyles)
	} else {
		opts.DateStyle, err = options.GetString("dateStyle", "", dateStyles)
	}

	if err != nil {
		return errorf(err)
	}

	timeStyles := oneOf("full", "long", "medium", "short")
	timePrecisions := oneOf("hour", "minute", "second")

	if _, ok := options["timePrecision"]; ok && options["timeStyle"] == nil {
		precision, err = options.GetString("timePrecision", "", timePrecisions)
		if err != nil {
			return errorf(err)
		}

		switch precision {
		case "second":
			opts.TimeStyle = "medium"
		case "minute", "hour":
			opts.TimeStyle = "short"
		}
	} else {
		opts.TimeStyle, err = options.GetString("timeStyle", "", timeStyles)
		if err != nil {
			return errorf(err)
		}
	}

	opts.TimeZone, err = getTZ(options)
	if err != nil {
		return errorf(err)
	}

	hourCycles := oneOf("h11", "h12", "h23", "h24")

	opts.HourCycle, err = options.GetString("hourCycle", "", hourCycles)
	if err != nil {
		return errorf(err)
	}

	dayPeriods := oneOf("short", "long")

	opts.DayPeriod, err = options.GetString("dayPeriod", "", dayPeriods)
	if err != nil {
		return errorf(err)
	}

	weekdays := oneOf("narrow", "short", "long")

	opts.Weekday, err = options.GetString("weekday", "", weekdays)
	if err != nil {
		return errorf(err)
	}

	eras := oneOf("narrow", "short", "long")

	opts.Era, err = options.GetString("era", "", eras)
	if err != nil {
		return errorf(err)
	}

	years := oneOf("numeric", "2-digit")

	opts.Year, err = options.GetString("year", "", years)
	if err != nil {
		return errorf(err)
	}

	months := oneOf("numeric", "2-digit", "narrow", "short", "long")

	opts.Month, err = options.GetString("month", "", months)
	if err != nil {
		return errorf(err)
	}

	days := oneOf("numeric", "2-digit")

	opts.Day, err = options.GetString("day", "", days)
	if err != nil {
		return errorf(err)
	}

	hours := oneOf("numeric", "2-digit")

	opts.Hour, err = options.GetString("hour", "", hours)
	if err != nil {
		return errorf(err)
	}

	minutes := oneOf("numeric", "2-digit")

	opts.Minute, err = options.GetString("minute", "", minutes)
	if err != nil {
		return errorf(err)
	}

	seconds := oneOf("numeric", "2-digit")

	opts.Second, err = options.GetString("second", "", seconds)
	if err != nil {
		return errorf(err)
	}

	const (
		minFractionalDigits = 1
		medFractionalDigits = 2
		maxFractionalDigits = 3
	)

	opts.FractionalSecondDigits, err = options.GetInt("fractionalSecondDigits", 0,
		oneOf(minFractionalDigits, medFractionalDigits, maxFractionalDigits))
	if err != nil {
		return errorf(err)
	}

	timeZoneNames := oneOf("long", "short", "shortOffset", "longOffset", "shortGeneric", "longGeneric")

	opts.TimeZoneName, err = options.GetString("timeZoneName", "", timeZoneNames)
	if err != nil {
		return errorf(err)
	}

	return &opts, nil
}

// datetimeFunc is the implementation of the datetime function. Locale-sensitive date and time formatting.
func datetimeFunc(operand *ResolvedValue, options Options, locale language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec datetime function: "+format, args...)
	}

	value, err := parseDatetimeOperand(operand)
	if err != nil {
		return errorf("%w", err)
	}

	opts, err := parseDatetimeOptions(options)
	if err != nil {
		return errorf("%w", err)
	}

	format := func() string {
		var layout string

		if opts.TimeZone != nil {
			value = value.In(opts.TimeZone)
		}

		if opts.Year != "" || opts.Day != "" {
			return intl.NewDateTimeFormat(locale, intl.Options{
				Year: intl.MustParseYear(opts.Year),
				Day:  intl.MustParseDay(opts.Day),
			}).Format(value)
		}

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
			case "full":
				layout += "15:04:05 MST"
			case "long":
				layout += "15:04:05 -0700"
			case "medium":
				layout += "15:04:05"
			case "short":
				layout += "15:04"
			}
		}

		return value.Format(layout)
	}

	return NewResolvedValue(value, WithFormat(format)), nil
}
