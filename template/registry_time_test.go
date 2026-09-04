package template

import (
	"errors"
	"testing"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

func Test_Time(t *testing.T) {
	t.Parallel()

	tests := []struct {
		options map[string]any
		wantErr error
		input   any
		name    string
		want    string
	}{
		{
			name:  "no options",
			input: testDate,
			want:  "03:04", // default style is "short"
		},
		{
			name:    "medium style",
			input:   testDate,
			options: map[string]any{"style": "medium"},
			want:    "03:04:05",
		},
		{
			name:    "long style",
			input:   testDate,
			options: map[string]any{"style": "long"},
			want:    "03:04:05 +0000",
		},
		{
			name:    "full style",
			input:   testDate,
			options: map[string]any{"style": "full"},
			want:    "03:04:05 UTC",
		},
		{
			name:    "precision second",
			input:   testDate,
			options: map[string]any{"precision": "second"},
			want:    "03:04:05",
		},
		{
			name:    "precision minute",
			input:   testDate,
			options: map[string]any{"precision": "minute"},
			want:    "03:04",
		},
		// errors
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
			name:    "illegal precision",
			input:   testDate,
			options: map[string]any{"precision": "invalid"},
			wantErr: mf2.ErrBadOption,
		},
		{
			name:    "unimplemented calendar",
			input:   testDate,
			options: map[string]any{"calendar": "buddhist"},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			opts := make(Options, len(test.options))
			for k, v := range test.options {
				opts[k] = NewResolvedValue(v)
			}

			v, err := timeFunc(NewResolvedValue(test.input), opts, language.AmericanEnglish)
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
