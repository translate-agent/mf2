package template

import (
	"errors"
	"testing"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

func Test_Date(t *testing.T) {
	t.Parallel()

	tests := []struct {
		options map[string]any
		wantErr error
		input   any
		locale  language.Tag
		name    string
		want    string
	}{
		{
			name:  "no options",
			input: testDate,
			want:  "1/2/2021", // default fields is "year-month-day", default length is "medium"
		},
		{
			name:    "length long",
			input:   testDate,
			options: map[string]any{"length": "long"},
			want:    "Jan/2/2021",
		},
		{
			name:    "length full",
			input:   testDate,
			options: map[string]any{"length": "full"},
			want:    "Jan/2/2021",
		},
		{
			name:    "style overrides length",
			input:   testDate,
			options: map[string]any{"style": "short", "length": "long"},
			want:    "1/2/21",
		},
		// fields option tests
		{
			name:    "fields weekday default",
			input:   testDate,
			options: map[string]any{"fields": "weekday"},
			want:    "Sat",
		},
		{
			name:    "fields weekday long",
			input:   testDate,
			options: map[string]any{"fields": "weekday", "length": "long"},
			want:    "Saturday",
		},
		{
			name:    "fields weekday short",
			input:   testDate,
			options: map[string]any{"fields": "weekday", "length": "short"},
			want:    "Sat",
		},
		{
			name:    "fields weekday french",
			input:   testDate,
			options: map[string]any{"fields": "weekday", "length": "long"},
			locale:  language.French,
			want:    "samedi",
		},
		{
			name:    "fields day-weekday long",
			input:   testDate,
			options: map[string]any{"fields": "day-weekday", "length": "long"},
			want:    "2 Saturday",
		},
		{
			name:    "fields day-weekday short",
			input:   testDate,
			options: map[string]any{"fields": "day-weekday", "length": "short"},
			want:    "2 Sat",
		},
		{
			name:    "fields day-weekday german",
			input:   testDate,
			options: map[string]any{"fields": "day-weekday", "length": "long"},
			locale:  language.German,
			want:    "Samstag, 2.",
		},
		{
			name:    "fields month-day long",
			input:   testDate,
			options: map[string]any{"fields": "month-day", "length": "long"},
			want:    "Jan/2",
		},
		{
			name:    "fields month-day medium",
			input:   testDate,
			options: map[string]any{"fields": "month-day", "length": "medium"},
			want:    "1/2",
		},
		{
			name:    "fields month-day short",
			input:   testDate,
			options: map[string]any{"fields": "month-day", "length": "short"},
			want:    "1/2",
		},
		{
			name:    "fields month-day latvian",
			input:   testDate,
			options: map[string]any{"fields": "month-day", "length": "long"},
			locale:  language.Latvian,
			want:    "02.01.",
		},
		{
			name:    "fields year-month-day long",
			input:   testDate,
			options: map[string]any{"fields": "year-month-day", "length": "long"},
			want:    "Jan/2/2021",
		},
		{
			name:    "fields year-month-day medium",
			input:   testDate,
			options: map[string]any{"fields": "year-month-day", "length": "medium"},
			want:    "1/2/2021",
		},
		{
			name:    "fields year-month-day short",
			input:   testDate,
			options: map[string]any{"fields": "year-month-day", "length": "short"},
			want:    "1/2/21",
		},
		{
			name:    "fields year-month-day latvian",
			input:   testDate,
			options: map[string]any{"fields": "year-month-day", "length": "long"},
			locale:  language.Latvian,
			want:    "2.01.2021.",
		},
		{
			name:    "fields with timeZone",
			input:   testDate, // 2021-01-02 03:04:05 UTC -> 2021-01-01 22:04:05 EST
			options: map[string]any{"fields": "weekday", "length": "long", "timeZone": "America/New_York"},
			want:    "Friday",
		},
		{
			name:    "nil operand",
			input:   nil,
			wantErr: mf2.ErrBadOperand,
		},
		{
			name:    "bad operand",
			input:   "testDate",
			wantErr: mf2.ErrBadOperand,
		},
		{
			name:    "illegal option",
			input:   testDate,
			options: map[string]any{"invalid": "option"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "illegal style",
			input:   testDate,
			options: map[string]any{"style": "invalid"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "illegal length",
			input:   testDate,
			options: map[string]any{"length": "invalid"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unimplemented calendar",
			input:   testDate,
			options: map[string]any{"calendar": "buddhist"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "invalid fields value",
			input:   testDate,
			options: map[string]any{"fields": "year"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unsupported fields month-day-weekday",
			input:   testDate,
			options: map[string]any{"fields": "month-day-weekday"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unsupported fields year-month-day-weekday",
			input:   testDate,
			options: map[string]any{"fields": "year-month-day-weekday"},
			wantErr: mf2.ErrBadOption,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := make(Options, len(test.options))
			for k, v := range test.options {
				opts[k] = NewResolvedValue(v)
			}

			loc := language.AmericanEnglish
			if test.locale != language.Und {
				loc = test.locale
			}

			v, err := dateFunc(NewResolvedValue(test.input), opts, loc)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Errorf("want %v, got %v", test.wantErr, err)
				}

				return
			}

			if err != nil {
				t.Error(err)

				return
			}

			got := v.format()
			if test.want != got {
				t.Errorf("want '%s', got '%s'", test.want, got)
			}
		})
	}
}

func Test_Date_NonLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		options Options
		wantErr error
		name    string
	}{
		{
			name: "non-literal fields",
			options: Options{
				"fields": &ResolvedValue{value: "weekday", isLiteral: false},
			},
			wantErr: mf2.ErrBadOption,
		},
		{
			name: "non-literal length",
			options: Options{
				"fields": NewResolvedValue("weekday"),
				"length": &ResolvedValue{value: "long", isLiteral: false},
			},
			wantErr: mf2.ErrBadOption,
		},
		{
			name: "non-literal style",
			options: Options{
				"style": &ResolvedValue{value: "short", isLiteral: false},
			},
			wantErr: mf2.ErrBadOption,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := dateFunc(NewResolvedValue(testDate), test.options, language.AmericanEnglish)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("want %v, got %v", test.wantErr, err)
			}
		})
	}
}

func Test_Date_Template(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		input   map[string]any
		locale  language.Tag
		want    string
		wantErr bool
	}{
		{
			name:    "fields weekday long",
			message: "Today is {$d :date fields=weekday length=long}",
			input:   map[string]any{"d": testDate},
			locale:  language.AmericanEnglish,
			want:    "Today is Saturday",
		},
		{
			name:    "fields weekday french",
			message: "Aujourd'hui est {$d :date fields=weekday length=long}",
			input:   map[string]any{"d": testDate},
			locale:  language.French,
			want:    "Aujourd'hui est samedi",
		},
		{
			name:    "fields month-day",
			message: "Date: {$d :date fields=month-day length=long}",
			input:   map[string]any{"d": testDate},
			locale:  language.AmericanEnglish,
			want:    "Date: Jan/2",
		},
		{
			name:    "non-literal fields variable error",
			message: ".local $f = {|weekday|} {{Today is {$d :date fields=$f}}}",
			input:   map[string]any{"d": testDate},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			tpl, err := New(WithLocale(test.locale)).Parse(test.message)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			got, err := tpl.Sprint(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got result: %s", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected execution error: %v", err)
			}

			if got != test.want {
				t.Errorf("want '%s', got '%s'", test.want, got)
			}
		})
	}
}

func Benchmark_DateFunc(b *testing.B) {
	opts := Options{
		"fields": NewResolvedValue("weekday"),
		"length": NewResolvedValue("long"),
	}
	resVal := NewResolvedValue(testDate)

	for b.Loop() {
		v, _ := dateFunc(resVal, opts, language.AmericanEnglish)
		_ = v.format()
	}
}

func Benchmark_DateFormat(b *testing.B) {
	opts := Options{
		"fields": NewResolvedValue("weekday"),
		"length": NewResolvedValue("long"),
	}
	resVal := NewResolvedValue(testDate)
	v, _ := dateFunc(resVal, opts, language.AmericanEnglish)

	for b.Loop() {
		_ = v.format()
	}
}
