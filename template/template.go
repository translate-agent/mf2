package template

import (
	"fmt"
	"io"
	"strings"
	"sync"

	ast "go.expect.digital/mf2/parse"
)

type execFn func(operand any, opts map[string]any) (string, error)

var execFuncs = map[string]execFn{}

var mutex sync.Mutex

// AddFunc adds a function to the template's function map.
func AddFunc(name string, f func(any, map[string]any) (string, error)) {
	mutex.Lock()
	execFuncs[name] = f
	mutex.Unlock()
}

type Template ast.AST

func New() *Template {
	return new(Template)
}

func Must(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}

	return t
}

func (t *Template) Parse(input string) (*Template, error) {
	ast, err := ast.Parse(input)
	if err != nil {
		return nil, syntaxErr(err)
	}

	*t = Template(ast)

	return t, nil
}

func (t *Template) Execute(wr io.Writer, input map[string]any) error {
	var resolve func() (string, error)

	switch message := t.Message.(type) {
	case nil:
		return nil
	case ast.SimpleMessage:
		resolve = func() (string, error) { return resolveSimpleMessage(message, input) }
	case ast.ComplexMessage:
		return fmt.Errorf("'%T' not implemented", message) // TODO: Implement.
	}

	s, err := resolve()
	if err != nil {
		return fmt.Errorf("resolve message: %w", err)
	}

	if _, err = wr.Write([]byte(s)); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// Sprint wraps Execute and returns the result as a string.
func (t *Template) Sprint(input map[string]any) (string, error) {
	sb := new(strings.Builder)

	if err := t.Execute(sb, input); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	return sb.String(), nil
}

// ------------------------------------Resolvers------------------------------------

func resolveSimpleMessage(message ast.SimpleMessage, input map[string]any) (string, error) {
	var s string

	for _, pattern := range message {
		switch pattern := pattern.(type) {
		case ast.TextPattern:
			s += string(pattern)
		case ast.Expression:
			expr, err := resolveExpression(pattern, input)
			if err != nil {
				return "", fmt.Errorf("resolve expression: %w", err)
			}

			s += expr
		case ast.Markup: // TODO: Implement.
			return "", fmt.Errorf("'%T' not implemented", pattern)
		}
	}

	return s, nil
}

func resolveExpression(expr ast.Expression, input map[string]any) (string, error) {
	value, err := resolveValue(expr.Operand, input)
	if err != nil {
		return "", fmt.Errorf("resolve value: %w", err)
	}

	if expr.Annotation == nil {
		return fmt.Sprint(value), nil // TODO: If value does not implement fmt.Stringer, what then ?
	}

	result, err := resolveAnnotation(value, expr.Annotation, input)
	if err != nil {
		return "", err
	}

	return result, nil
}

// resolveValue resolves the value of an expression's operand.
//
//   - If the operand is a literal, it returns the literal's value.
//   - If the operand is a variable, it returns the value of the variable from the input map.
func resolveValue(v ast.Value, input map[string]any) (any, error) {
	var resolved any

	switch v := v.(type) {
	case nil:
		// noop
	case ast.QuotedLiteral:
		resolved = string(v)
	case ast.NameLiteral:
		resolved = string(v)
	case ast.NumberLiteral:
		resolved = float64(v)
	case ast.Variable:
		val, ok := input[string(v)]
		if !ok {
			return nil, unresolvedVariableErr(v)
		}

		resolved = val
	}

	return resolved, nil
}

func resolveAnnotation(operand any, annotation ast.Annotation, input map[string]any) (string, error) {
	annoFn, ok := annotation.(ast.Function)
	if !ok {
		return "", unsupportedExpressionErr(annotation)
	}

	execF, ok := execFuncs[annoFn.Identifier.Name]
	if !ok {
		return "", unknownFunctionErr(annoFn.Identifier.Name)
	}

	opts, err := resolveOptions(annoFn.Options, input)
	if err != nil {
		return "", fmt.Errorf("resolve options: %w", err)
	}

	result, err := execF(operand, opts)
	if err != nil {
		return "", formattingErr(err)
	}

	return result, nil
}

func resolveOptions(options []ast.Option, input map[string]any) (map[string]any, error) {
	m := make(map[string]any, len(options))

	for _, opt := range options {
		name := opt.Identifier.Name
		if _, ok := m[name]; ok {
			return nil, duplicateOptionNameErr(name)
		}

		value, err := resolveValue(opt.Value, input)
		if err != nil {
			return nil, fmt.Errorf("resolve value: %w", err)
		}

		m[name] = value
	}

	return m, nil
}
