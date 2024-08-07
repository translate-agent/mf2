package template

import (
	"testing"

	"golang.org/x/text/language"
)

func Test_Time(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   any
		options map[string]any
		name    string
		want    string
		wantErr bool
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
		// errors
		{
			name:    "nil operand",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "bad operand",
			input:   "testDate",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			v, err := timeFunc(NewResolvedValue(test.input), test.options, language.AmericanEnglish)
			if test.wantErr {
				if err == nil {
					t.Error("want error, got nil")
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
