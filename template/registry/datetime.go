package registry

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/language"
)

// https://github.com/unicode-org/message-format-wg/blob/20a61b4af534acb7ecb68a3812ca0143b34dfc76/spec/registry.xml#L13

// datetimeFunc is the implementation of the datetime function. Locale-sensitive date and time formatting.
var datetimeRegistryFunc = &Func{
	Name:           "datetime",
	Func:           datetimeFunc,
	MatchSignature: nil, // Not allowed to use in matching context
	FormatSignature: &Signature{
		IsInputRequired: true,
		ValidateInput: func(a any) error {
			if _, ok := a.(time.Time); !ok {
				return fmt.Errorf("unsupported type: %T", a)
			}

			return nil
		},
		Options: Options{
			{
				// The predefined date formatting style to use.
				Name:           "dateStyle",
				PossibleValues: []any{"full", "long", "medium", "short"},
			},
			{
				// The predefined time formatting style to use.
				Name:           "timeStyle",
				PossibleValues: []any{"full", "long", "medium", "short"},
			},
			{
				//  Calendar to use.
				Name: "calendar",
				PossibleValues: []any{
					"buddhist", "chinese", "coptic", "dangi", "ethioaa", "ethiopic", "gregory",
					"hebrew", "indian", "islamic", "islamic-umalqura", "islamic-tbla",
					"islamic-civil", "islamic-rgsa", "iso8601", "japanese", "persian", "roc",
				},
			},
			{
				// Numbering system to use.
				Name: "numberingSystem",
				PossibleValues: []any{
					"arab", "arabext", "bali", "beng", "deva", "fullwide", "gujr", "guru", "hanidec",
					"khmr", "knda", "laoo", "latn", "limb", "mlym", "mong", "mymr", "orya", "tamldec",
					"telu", "thai", "tibt",
				},
			},
			{
				// The time zone to use.
				// The only value implementations must recognize is "UTC";
				// the default is the runtime's default time zone.
				// Implementations may also recognize the time zone names of the IANA time zone database,
				// such as "Asia/Shanghai", "Asia/Kolkata", "America/New_York".
				Name:    "timeZone",
				Default: "UTC",
			},
			{
				// The hour cycle to use.
				Name:           "hourCycle",
				PossibleValues: []any{"h11", "h12", "h23", "h24"},
			},
			{
				// The formatting style used for day periods like "in the morning", "am", "noon", "n" etc.
				Name:           "dayPeriod",
				PossibleValues: []any{"narrow", "short", "long"},
			},
			{
				// The representation of the weekday.
				Name:           "weekday",
				PossibleValues: []any{"narrow", "short", "long"},
			},
			{
				// The representation of the era.
				Name:           "era",
				PossibleValues: []any{"narrow", "short", "long"},
			},
			{
				// The representation of the year.
				Name:           "year",
				PossibleValues: []any{"numeric", "2-digit"},
			},
			{
				// The representation of the month.
				Name:           "month",
				PossibleValues: []any{"numeric", "2-digit", "narrow", "short", "long"},
			},
			{
				// The representation of the day.
				Name:           "day",
				PossibleValues: []any{"numeric", "2-digit"},
			},
			{
				// The representation of the hour.
				Name:           "hour",
				PossibleValues: []any{"numeric", "2-digit"},
			},
			{
				// The representation of the minute.
				Name:           "minute",
				PossibleValues: []any{"numeric", "2-digit"},
			},
			{
				// The representation of the second.
				Name:           "second",
				PossibleValues: []any{"numeric", "2-digit"},
			},
			{
				// The number of digits used to represent fractions of a second (any additional digits are truncated).
				Name:           "fractionalSecondDigits",
				PossibleValues: []any{1, 2, 3},
			},
			{
				// The localized representation of the time zone name.
				Name:           "timeZoneName",
				PossibleValues: []any{"long", "short", "shortOffset", "longOffset", "shortGeneric", "longGeneric"},
			},
		},
	},
}

func datetimeFunc(a any, options map[string]any, locale language.Tag) (any, error) {
	if len(options) == 0 {
		return fmt.Sprint(a), nil
	}

	tim := a.(time.Time) //nolint:forcetypeassert // we already validated the input type

	// TODO: implement all options
	for optName := range options {
		switch optName {
		case "calendar", "numberingSystem", "hourCycle", "dayPeriod", "weekday", "era",
			"year", "month", "day", "hour", "minute", "second", "fractionalSecondDigits":
			return nil, fmt.Errorf("option '%s' is not implemented", optName)
		}
	}

	var layout string

	if dateStyle, ok := options["dateStyle"].(string); ok {
		switch dateStyle {
		case "full":
			layout = "Monday, 02 January 2006"
		case "long":
			layout = "02 January 2006"
		case "medium":
			layout = "02 Jan 2006"
		case "short":
			layout = "02/01/06"
		}
	}

	if timeStyle, ok := options["timeStyle"].(string); ok {
		switch timeStyle {
		case "full", "long":
			layout += " 15:04:05"
		case "medium":
			layout += " 15:04"
		case "short":
			layout += " 15"
		}
	}

	layout = strings.TrimSpace(layout)

	if timeZone, ok := options["timeZone"].(string); ok {
		loc, err := time.LoadLocation(timeZone)
		if err != nil {
			return nil, fmt.Errorf("load location: %w", err)
		}

		tim = tim.In(loc)
	}

	return tim.Format(layout), nil
}
