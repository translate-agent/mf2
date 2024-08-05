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

			got, err := dateFunc(test.input, test.options, language.AmericanEnglish)
			if v, ok := got.(*ResolvedValue); ok {
				got = v.format()
			}

			if test.wantErr {
				if err == nil {
					t.Error("want error, got nil")
				}

				return
			}

			if err != nil {
				t.Error(err)
			}

			if test.want != got {
				t.Errorf("want '%s', got '%s'", test.want, got)
			}
		})
	}
}
