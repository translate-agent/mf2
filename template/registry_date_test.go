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
		name    string
		want    string
	}{
		{
			name:  "no options",
			input: testDate,
			want:  "02/01/21", // default style is "short"
		},
		{
			name:    "length long",
			input:   testDate,
			options: map[string]any{"length": "long"},
			want:    "02 January 2021",
		},
		{
			name:    "length full",
			input:   testDate,
			options: map[string]any{"length": "full"},
			want:    "Saturday, 02 January 2021",
		},
		{
			name:    "style overrides length",
			input:   testDate,
			options: map[string]any{"style": "short", "length": "long"},
			want:    "02/01/21",
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
			name:    "unimplemented fields",
			input:   testDate,
			options: map[string]any{"fields": "year"},
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

			v, err := dateFunc(NewResolvedValue(test.input), opts, language.AmericanEnglish)
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
