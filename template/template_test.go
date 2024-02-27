package template

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fn struct {
	f    Func
	name string
}

func Test_ExecuteSimpleMessage(t *testing.T) {
	t.Parallel()

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

			template, err := New().Parse(tt.args.inputStr)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f.name, f.f)
			}

			actual, err := template.Sprint(tt.args.inputMap)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_ExecuteComplexMessage(t *testing.T) {
	t.Parallel()

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
			name: "complex message without declaration",
			args: args{
				inputStr: "{{Hello, {|literal|} World!}}",
				inputMap: nil,
			},
			expected: "Hello, literal World!",
		},
		{
			name: "local declarations",
			args: args{
				inputStr: `.local $var1 = { literalExpression }
		.local $var2 = { $anotherVar }
		.local $var3 = { :randNum }
		{{Hello, {$var1} {$var2} {$var3}!}}`,
				inputMap: map[string]any{"anotherVar": "World"},
			},
			funcs: []fn{
				{
					name: "randNum",
					f: func(_ any, _ map[string]any) (string, error) {
						return "0", nil
					},
				},
			},
			expected: "Hello, literalExpression World 0!",
		},
		{
			name: "input declaration",
			args: args{
				inputStr: ".input { $name :upper } {{Hello, {$name}!}}",
				inputMap: map[string]any{"name": "john"},
			},
			funcs: []fn{
				{
					name: "upper",
					f: func(v any, _ map[string]any) (string, error) {
						return strings.ToUpper(fmt.Sprint(v)), nil
					},
				},
			},
			expected: "Hello, JOHN!",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.args.inputStr)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f.name, f.f)
			}

			actual, err := template.Sprint(tt.args.inputMap)
			require.NoError(t, err)

			require.Equal(t, tt.expected, actual)
		})
	}
}

func Test_ExecuteErrors(t *testing.T) {
	t.Parallel()

	type args struct {
		inputMap map[string]any
		inputStr string
	}

	tests := []struct {
		name               string
		expectedParseErr   error
		expectedExecuteErr error
		args               args
		fn                 fn // exec function to be added before executing
	}{
		{
			name: "syntax error",
			args: args{
				inputStr: "Hello { $name",
				inputMap: nil,
			},
			expectedParseErr: ErrSyntax,
		},
		{
			name: "unresolved variable",
			args: args{
				inputStr: "Hello, { $name }!",
				inputMap: nil,
			},
			expectedExecuteErr: ErrUnresolvedVariable,
		},
		{
			name: "unknown function",
			args: args{
				inputStr: "Hello, { :f }!",
				inputMap: nil,
			},
			expectedExecuteErr: ErrUnknownFunction,
		},
		{
			name: "duplicate option name",
			args: args{
				inputStr: "Hello, { :existing style=first style=second }!",
				inputMap: nil,
			},
			expectedExecuteErr: ErrDuplicateOptionName,
			fn: fn{
				name: "existing",
				f:    func(any, map[string]any) (string, error) { return "", nil },
			},
		},
		{
			name: "unsupported expression",
			args: args{
				inputStr: "Hello, { 12 ^private }!",
				inputMap: nil,
			},
			expectedExecuteErr: ErrUnsupportedExpression,
		},
		{
			name: "formatting error",
			args: args{
				inputStr: "Hello, { :error }!",
				inputMap: nil,
			},
			expectedExecuteErr: ErrFormatting,
			fn: fn{
				name: "error",
				f:    func(any, map[string]any) (string, error) { return "", fmt.Errorf("error occurred") },
			},
		},
		{
			name: "unsupported declaration",
			args: args{
				inputStr: ".reserved { $name } {{Hello, {$name}!}}",
				inputMap: nil,
			},
			expectedExecuteErr: ErrUnsupportedStatement,
		},
		{
			name: "duplicate declaration",
			args: args{
				inputStr: ".input {$var} .input {$var} {{Redeclaration of the same variable}}",
				inputMap: map[string]any{"var": "22"},
			},
			expectedExecuteErr: ErrDuplicateDeclaration,
		},
		{
			name: "duplicate declaration",
			args: args{
				inputStr: ".local $var = {$ext} .input {$var} {{Redeclaration of a local variable}}",
				inputMap: map[string]any{"ext": "22"},
			},
			expectedExecuteErr: ErrDuplicateDeclaration,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.args.inputStr)
			if tt.expectedParseErr != nil {
				require.ErrorIs(t, err, tt.expectedParseErr)
				return
			}

			template.AddFunc(tt.fn.name, tt.fn.f)

			_, err = template.Sprint(tt.args.inputMap)
			require.ErrorIs(t, err, tt.expectedExecuteErr)
		})
	}
}
