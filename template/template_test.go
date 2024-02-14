package template

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_ExecuteSimpleMessage(t *testing.T) {
	t.Parallel()

	type fn struct {
		f    execFn
		name string
	}

	type args struct {
		inputMap map[string]any
		inputStr string
	}

	tests := []struct {
		name     string
		args     args
		expected string
		funcs    []fn // exec functions to be added before executing
	}{
		{
			name: "empty message",
			args: args{
				inputStr: "",
				inputMap: nil,
			},
			expected: "",
		},
		{
			name: "plain message",
			args: args{
				inputStr: "Hello, World!",
				inputMap: nil,
			},
			expected: "Hello, World!",
		},
		{
			name: "variables and literals",
			args: args{
				inputStr: "Hello, { $name } { unquoted } { |quoted| } { 42 }!",
				inputMap: map[string]any{"name": "World"},
			},
			expected: "Hello, World unquoted quoted 42!",
		},
		{
			name: "functions with operand",
			args: args{
				inputStr: "Hello, { $firstName :upper } { $secondName :lower style=first } { $date :date }!",
				inputMap: map[string]any{
					"firstName":  "John",
					"secondName": "Doe",
					"date":       time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			funcs: []fn{
				{
					name: "upper",
					f: func(v any, _ map[string]any) (string, error) {
						return strings.ToUpper(fmt.Sprint(v)), nil
					},
				},
				{
					name: "lower",
					f: func(v any, opts map[string]any) (string, error) {
						if opts == nil {
							return strings.ToLower(fmt.Sprint(v)), nil
						}

						if style, ok := opts["style"].(string); ok {
							switch style {
							case "first":
								return strings.ToLower(fmt.Sprint(v)[0:1]) + fmt.Sprint(v)[1:], nil
							default:
								return "", fmt.Errorf("unsupported style: %s", style)
							}
						}

						return "", nil
					},
				},
				{
					name: "date",
					f: func(v any, _ map[string]any) (string, error) {
						date, ok := v.(time.Time)
						if !ok {
							return "", fmt.Errorf("unsupported type: %T", v)
						}

						return date.Format("2006-01-02"), nil
					},
				},
			},
			expected: "Hello, JOHN doe 2021-09-01!",
		},
		{
			name: "function without operand",
			args: args{
				inputStr: "Hello, { :randFirstName } { :randSecondName style=caps }!",
				inputMap: nil,
			},
			funcs: []fn{
				{
					name: "randFirstName",
					f: func(_ any, _ map[string]any) (string, error) {
						return "John", nil
					},
				},
				{
					name: "randSecondName",
					f: func(_ any, opts map[string]any) (string, error) {
						if opts == nil {
							return "Doe", nil
						}

						if style, ok := opts["style"].(string); ok {
							switch style {
							case "caps":
								return "DOE", nil
							default:
								return "", fmt.Errorf("unsupported style: %s", style)
							}
						}

						return "", nil
					},
				},
			},
			expected: "Hello, John DOE!",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, f := range tt.funcs {
				AddFunc(f.name, f.f)
			}

			template, err := New().Parse(tt.args.inputStr)
			require.NoError(t, err)

			actual, err := template.Sprint(tt.args.inputMap)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_ExecuteErrors(t *testing.T) {
	t.Parallel()

	AddFunc("existing", func(any, map[string]any) (string, error) { return "", nil })
	AddFunc("error", func(any, map[string]any) (string, error) { return "", fmt.Errorf("error") })

	tests := []struct {
		name               string
		expectedParseErr   error
		expectedExecuteErr error
		inputStr           string
	}{
		{
			name:             "syntax error",
			inputStr:         "Hello { $name",
			expectedParseErr: ErrSyntax,
		},
		{
			name:               "unresolved variable",
			inputStr:           "Hello, { $name }!",
			expectedExecuteErr: ErrUnresolvedVariable,
		},
		{
			name:               "unknown function",
			inputStr:           "Hello, { :f }!",
			expectedExecuteErr: ErrUnknownFunction,
		},
		{
			name:               "duplicate option name",
			inputStr:           "Hello, { :existing style=first style=second }!",
			expectedExecuteErr: ErrDuplicateOptionName,
		},
		{
			name:               "unsupported expression",
			inputStr:           "Hello, { 12 ^private }!",
			expectedExecuteErr: ErrUnsupportedExpression,
		},
		{
			name:               "formatting error",
			inputStr:           "Hello, { :error }!",
			expectedExecuteErr: ErrFormatting,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.inputStr)
			if tt.expectedParseErr != nil {
				require.ErrorIs(t, err, tt.expectedParseErr)
				return
			}

			_, err = template.Sprint(nil)
			require.ErrorIs(t, err, tt.expectedExecuteErr)
		})
	}
}
