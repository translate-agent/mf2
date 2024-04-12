package registry

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/text/language"
)

// https://github.com/unicode-org/message-format-wg/blob/20a61b4af534acb7ecb68a3812ca0143b34dfc76/spec/registry.xml#L13

// datetimeFunc is the implementation of the datetime function. Locale-sensitive date and time formatting.
var datetimeRegistryFunc = F{
	Format: datetimeFunc,
}

type datetimeOptions struct {
	TimeZone               *time.Location
	DateStyle              string
	TimeStyle              string
	Calendar               string
	NumberingSystem        string
	HourCycle              string
	DayPeriod              string
	Weekday                string
	Era                    string
	Year                   string
	Month                  string
	Day                    string
	Hour                   string
	Minute                 string
	Second                 string
	TimeZoneName           string
	FractionalSecondDigits int
}

func parseDatetimeInput(input any) (time.Time, error) {
	if input == nil {
		return time.Time{}, errors.New("input is required, got nil")
	}

	v, ok := input.(time.Time)
	if !ok {
		return time.Time{}, fmt.Errorf("unsupported type: %T", input)
	}

	return v, nil
}

func parseDatetimeOptions(options Opts) (*datetimeOptions, error) {
	var (
		opts datetimeOptions
		err  error
	)

	// The predefined date formatting style to use.
	dateStyles := []string{"full", "long", "medium", "short"}
	if opts.DateStyle, err = options.GetString("dateStyle", "", dateStyles); err != nil {
		return nil, err
	}

	// The predefined time formatting style to use.
	timeStyles := []string{"full", "long", "medium", "short"}
	if opts.TimeStyle, err = options.GetString("timeStyle", "", timeStyles); err != nil {
		return nil, err
	}

	//  Calendar to use.
	calendars := []string{
		"buddhist", "chinese", "coptic", "dangi", "ethioaa", "ethiopic", "gregory",
		"hebrew", "indian", "islamic", "islamic-umalqura", "islamic-tbla",
		"islamic-civil", "islamic-rgsa", "iso8601", "japanese", "persian", "roc",
	}
	if opts.Calendar, err = options.GetString("calendar", "", calendars); err != nil {
		return nil, err
	}

	// Numbering system to use.
	numberingSystems := []string{
		"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru", "hanidec",
		"khmr", "knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec",
		"telu", "thai", "tibt",
	}
	if opts.NumberingSystem, err = options.GetString("numberingSystem", "", numberingSystems); err != nil {
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

	// The hour cycle to use.
	hourCycles := []string{"h11", "h12", "h23", "h24"}
	if opts.HourCycle, err = options.GetString("hourCycle", "", hourCycles); err != nil {
		return nil, err
	}

	// The formatting style used for day periods like "in the morning", "am", "noon", "n" etc.
	dayPeriods := []string{"short", "long"}
	if opts.DayPeriod, err = options.GetString("dayPeriod", "", dayPeriods); err != nil {
		return nil, err
	}

	// The representation of the weekday.
	weekdays := []string{"narrow", "short", "long"}
	if opts.Weekday, err = options.GetString("weekday", "", weekdays); err != nil {
		return nil, err
	}

	// The representation of the era.
	eras := []string{"narrow", "short", "long"}
	if opts.Era, err = options.GetString("era", "", eras); err != nil {
		return nil, err
	}

	// The representation of the year.
	years := []string{"numeric", "2-digit"}
	if opts.Year, err = options.GetString("year", "", years); err != nil {
		return nil, err
	}

	// The representation of the month.
	months := []string{"numeric", "2-digit", "narrow", "short", "long"}
	if opts.Month, err = options.GetString("month", "", months); err != nil {
		return nil, err
	}

	// The representation of the day.
	days := []string{"numeric", "2-digit"}
	if opts.Day, err = options.GetString("day", "", days); err != nil {
		return nil, err
	}

	// The representation of the hour.
	hours := []string{"numeric", "2-digit"}
	if opts.Hour, err = options.GetString("hour", "", hours); err != nil {
		return nil, err
	}

	// The representation of the minute.
	minutes := []string{"numeric", "2-digit"}
	if opts.Minute, err = options.GetString("minute", "", minutes); err != nil {
		return nil, err
	}

	// The representation of the second.
	seconds := []string{"numeric", "2-digit"}
	if opts.Second, err = options.GetString("second", "", seconds); err != nil {
		return nil, err
	}

	// The number of fractional seconds to display.
	if opts.FractionalSecondDigits, err = options.GetPositiveInt("fractionalSecondDigits", 0, []int{1, 2, 3}); err != nil {
		return nil, err
	}

	// The localized representation of the time zone name.
	timeZoneNames := []string{"long", "short", "shortOffset", "longOffset", "shortGeneric", "longGeneric"}
	if opts.TimeZoneName, err = options.GetString("timeZoneName", "", timeZoneNames); err != nil {
		return nil, err
	}

	return &opts, nil
}

func datetimeFunc(input any, options Opts, locale language.Tag) (any, error) {
	tim, err := parseDatetimeInput(input)
	if err != nil {
		return "", err
	}

	opts, err := parseDatetimeOptions(options)
	if err != nil {
		return "", err
	}

	if len(options) == 0 {
		return fmt.Sprint(input), nil
	}

	for optName := range options {
		switch optName {
		case "calendar", "numberingSystem", "hourCycle", "dayPeriod", "weekday", "era",
			"year", "month", "day", "hour", "minute", "second", "fractionalSecondDigits":
			return nil, fmt.Errorf("option '%s' is not implemented", optName)
		}
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

	tim = tim.In(opts.TimeZone)

	return tim.Format(layout), nil
}
