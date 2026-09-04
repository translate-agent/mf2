package template

import (
	"errors"
	"testing"
	"time"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

var testDate = time.Date(2021, 1, 2, 3, 4, 5, 6, time.UTC)

func Test_Datetime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		options map[string]any
		wantErr error
		input   any
		name    string
		want    string
	}{
		// positive tests
		{
			// {$d :datetime} is the same as {$d :datetime dateStyle=medium timeStyle=short}
			name:  "no options",
			input: testDate,
			want:  "02 Jan 2021 03:04",
		},
		{
			name:    "dateStyle",
			input:   testDate,
			options: map[string]any{"dateStyle": "full"},
			want:    "Saturday, 02 January 2021",
		},
		{
			name:    "dateLength",
			input:   testDate,
			options: map[string]any{"dateLength": "full"},
			want:    "Saturday, 02 January 2021",
		},
		{
			name:    "dateStyle overrides dateLength",
			input:   testDate,
			options: map[string]any{"dateStyle": "short", "dateLength": "full"},
			want:    "02/01/21",
		},
		{
			name:    "dateLength and timePrecision",
			input:   testDate,
			options: map[string]any{"dateLength": "short", "timePrecision": "second"},
			want:    "02/01/21 03:04:05",
		},
		{
			name:    "timeStyle",
			input:   testDate,
			options: map[string]any{"timeStyle": "medium"},
			want:    "03:04:05",
		},
		{
			name:    "dateStyle and timeStyle",
			input:   testDate,
			options: map[string]any{"dateStyle": "short", "timeStyle": "long"},
			want:    "02/01/21 03:04:05 +0000",
		},
		{
			name:    "timeZone",
			input:   testDate,
			options: map[string]any{"timeStyle": "long", "dateStyle": "medium", "timeZone": "EET"},
			want:    "02 Jan 2021 05:04:05 +0200",
		},
		{
			name:    "2 digit year",
			input:   testDate,
			options: map[string]any{"year": "2-digit"},
			want:    "21",
		},
		{
			name:    "numeric year",
			input:   testDate,
			options: map[string]any{"year": "numeric"},
			want:    "2021",
		},
		{
			name:    "2 digit day",
			input:   testDate,
			options: map[string]any{"day": "2-digit"},
			want:    "02",
		},
		{
			name:    "timePrecision second",
			input:   testDate,
			options: map[string]any{"timePrecision": "second"},
			want:    "03:04:05",
		},
		{
			name:    "timePrecision minute",
			input:   testDate,
			options: map[string]any{"timePrecision": "minute"},
			want:    "03:04",
		},
		// negative tests
		{
			name:    "not implemented",
			input:   testDate,
			options: map[string]any{"calendar": "buddhist"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unimplemented dateFields",
			input:   testDate,
			options: map[string]any{"dateFields": "month-day"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "illegal timePrecision",
			input:   testDate,
			options: map[string]any{"timePrecision": "invalid"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "illegal dateLength",
			input:   testDate,
			options: map[string]any{"dateLength": "invalid"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unimplemented hour12",
			input:   testDate,
			options: map[string]any{"hour12": true},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unimplemented timeZoneStyle",
			input:   testDate,
			options: map[string]any{"timeZoneStyle": "short"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "illegal option",
			input:   testDate,
			options: map[string]any{"invalid": "option"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "illegal type",
			input:   struct{}{},
			options: nil,
			wantErr: mf2.ErrBadOperand,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := make(Options, len(test.options))
			for k, v := range test.options {
				opts[k] = NewResolvedValue(v)
			}

			v, err := datetimeFunc(NewResolvedValue(test.input), opts, language.AmericanEnglish)
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
