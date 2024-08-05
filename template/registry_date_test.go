package template

import (
	"testing"

	"golang.org/x/text/language"
)

func Test_Date(t *testing.T) {
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
			want:  "02/01/21", // default style is "short"
		},
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

			v, err := dateFunc(test.input, test.options, language.AmericanEnglish)

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
