package template

import (
	"testing"
	"time"

	"golang.org/x/text/language"
)

func Test_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		options map[string]any
		want    string
		wantErr bool
	}{
		// positive
		{
			name:    "int",
			input:   53,
			options: nil,
			want:    "53",
		},
		{
			name:    "date",
			input:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			options: nil,
			want:    "2021-01-01 00:00:00 +0000 UTC",
		},
		// negative
		{
			name:    "illegal type", // does not implement stringer, and is not castable to string
			input:   struct{}{},
			options: nil,
			want:    "",
		},
		{
			name:    "illegal options", // string function does not support any options
			input:   2,
			options: map[string]any{"will": "fail"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			v, err := stringFunc(test.input, test.options, language.AmericanEnglish)
			if test.wantErr {
				if err == nil {
					t.Error("want error, got nil")
				}

				// TODO(jhorsts): assert got value

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
