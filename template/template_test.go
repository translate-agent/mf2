package template

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.expect.digital/mf2/template/registry"
)

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
		funcs    []registry.Func // format functions to be added before executing
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
				inputStr: "Hello, { $firstName :string } your age is { $age :number style=decimal }!",
				inputMap: map[string]any{
					"firstName": "John",
					"age":       23,
				},
			},
			expected: "Hello, John your age is 23!",
		},
		{
			name: "function without operand",
			args: args{
				inputStr: "Hello, { :randName }",
				inputMap: nil,
			},
			funcs: []registry.Func{
				{
					Name:            "randName",
					FormatSignature: &registry.Signature{},
					Fn:              func(_ any, _ map[string]any) (any, error) { return "John", nil },
				},
			},
			expected: "Hello, John",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.args.inputStr)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f)
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
		funcs    []registry.Func // format functions to be added before executing
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
			funcs: []registry.Func{
				{
					Name:            "randNum",
					FormatSignature: &registry.Signature{},
					Fn:              func(_ any, _ map[string]any) (any, error) { return 0, nil },
				},
			},
			expected: "Hello, literalExpression World 0!",
		},
		{
			name: "input declaration",
			args: args{
				inputStr: ".input { $name :string } {{Hello, {$name}!}}",
				inputMap: map[string]any{"name": 999},
			},
			expected: "Hello, 999!",
		},
		{
			name: "markup",
			args: args{
				inputStr: "Click {#link href=$url}here{/link} standalone {#foo/}",
				inputMap: nil,
			},
			expected: "Click here standalone ",
		},
		{
			name: "matcher many",
			args: args{
				inputStr: ".match { $n } 0 {{no apples}} 1 {{{ $n } apple}} * {{{ $n } apples}}",
				inputMap: map[string]any{"n": "9898"},
			},
			expected: "9898 apples",
		},
		{
			name: "matcher one",
			args: args{
				inputStr: ".match { $n } 0 {{no apples}} 1 {{{ $n } apple}} * {{{ $n } apples}}",
				inputMap: map[string]any{"n": "1"},
			},
			expected: "1 apple",
		},
		{
			name: "matcher no",
			args: args{
				inputStr: ".match { $n } 0 {{no apples}} 1 {{{ $n } apple}} * {{{ $n } apples}}",
				inputMap: map[string]any{"n": "0"},
			},
			expected: "no apples",
		},
		{
			name: "Pattern Selection with string annotation",
			args: args{
				//nolint:dupword
				inputStr: ".match {$foo :string} {$bar :string} bar bar {{All bar}} foo foo {{All foo}} * * {{Otherwise}}",
				inputMap: map[string]any{"foo": "foo", "bar": "bar"},
			},
			expected: "Otherwise",
		},
		{
			name: "Pattern Selection with Multiple Variants",
			args: args{
				inputStr: `.match {$foo :string} {$bar :string} * bar {{Any and bar}}
							foo * {{Foo and any}} foo bar {{Foo and bar}} * * {{Otherwise}}`,
				inputMap: map[string]any{"foo": "foo", "bar": "bar"},
			},
			expected: "Foo and bar",
		},
		{
			name: "Plural Format Selection",
			args: args{
				inputStr: ".match {$count :number} one {{Category match}} 1 {{Exact match}} *   {{Other match}}",
				inputMap: map[string]any{"count": "1"},
			},
			expected: "Exact match",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template, err := New().Parse(tt.args.inputStr)
			require.NoError(t, err)

			for _, f := range tt.funcs {
				template.AddFunc(f)
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
		fn                 []registry.Func // format function to be added before executing
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
				inputStr: "Hello, { :number style=decimal style=percent }!",
				inputMap: nil,
			},
			expectedExecuteErr: ErrDuplicateOptionName,
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
			fn: []registry.Func{
				{
					Name:            "error",
					FormatSignature: &registry.Signature{},
					Fn:              func(any, map[string]any) (any, error) { return nil, errors.New("error") },
				},
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
		{
			name: "selection error",
			args: args{
				inputStr: ".match {$n} 0 {{no apples}} 1 {{apple}} * {{apples}}",
				inputMap: map[string]any{"": ""},
			},
			expectedExecuteErr: ErrSelection,
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

			for _, f := range tt.fn {
				template.AddFunc(f)
			}

			_, err = template.Sprint(tt.args.inputMap)
			require.ErrorIs(t, err, tt.expectedExecuteErr)
		})
	}
}
