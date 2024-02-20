package template

import (
	"errors"
	"fmt"
	"io"
	"strings"

	ast "go.expect.digital/mf2/parse"
)

// MessageFormat2 Errors as defined in the specification.
//
// https://github.com/unicode-org/message-format-wg/blob/122e64c2482b54b6eff4563120915e0f86de8e4d/spec/errors.md
var (
	ErrSyntax                = errors.New("syntax error")
	ErrUnresolvedVariable    = errors.New("unresolved variable")
	ErrUnknownFunction       = errors.New("unknown function reference")
	ErrDuplicateOptionName   = errors.New("duplicate option name")
	ErrUnsupportedExpression = errors.New("unsupported expression")
	ErrFormatting            = errors.New("formatting error")
)

// Func is a function, that will be called when a function is encountered in the template.
type Func func(operand any, options map[string]any) (string, error)

// Template represents a MessageFormat2 template.
type Template struct {
	ast   *ast.AST
	funcs map[string]Func
}

// AddFunc adds a function to the template's function map.
func (t *Template) AddFunc(name string, f Func) {
	t.funcs[name] = f
}

// Parse parses the MessageFormat2 string and returns the template.
func (t *Template) Parse(input string) (*Template, error) {
	ast, err := ast.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSyntax, err.Error())
	}

	t.ast = &ast

	return t, nil
}

// Execute writes the result of the template to the given writer.
func (t *Template) Execute(wr io.Writer, input map[string]any) error {
	if t.ast == nil {
		return errors.New("AST is nil")
	}

	executer := &executer{template: t, wr: wr, input: input}

	return executer.execute()
}

// Sprint wraps Execute and returns the result as a string.
func (t *Template) Sprint(input map[string]any) (string, error) {
	sb := new(strings.Builder)

	if err := t.Execute(sb, input); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	return sb.String(), nil
}

// New returns a new Template.
func New() *Template { return &Template{funcs: make(map[string]Func)} }

type executer struct {
	template *Template
	wr       io.Writer
	input    map[string]any
}

func (e *executer) write(s string) error {
	if _, err := e.wr.Write([]byte(s)); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (e *executer) execute() error {
	switch message := e.template.ast.Message.(type) {
	default:
		return fmt.Errorf("unknown message type: '%T'", message)
	case nil:
		return nil
	case ast.SimpleMessage:
		return e.resolveSimpleMessage(message)
	case ast.ComplexMessage:
		return errors.New("complex message not implemented") // TODO: Implement.
	}
}

func (e *executer) resolveSimpleMessage(message ast.SimpleMessage) error {
	for _, pattern := range message {
		switch pattern := pattern.(type) {
		case ast.TextPattern:
			if err := e.write(string(pattern)); err != nil {
				return err
			}
		case ast.Expression:
			if err := e.resolveExpression(pattern); err != nil {
				return fmt.Errorf("resolve expression: %w", err)
			}
		case ast.Markup: // TODO: Implement.
			return fmt.Errorf("'%T' not implemented", pattern)
		}
	}

	return nil
}

func (e *executer) resolveExpression(expr ast.Expression) error {
	value, err := e.resolveValue(expr.Operand)
	if err != nil {
		return fmt.Errorf("resolve value: %w", err)
	}

	if expr.Annotation == nil {
		// NOTE: Parser won't allow value to be nil if annotation is nil.
		return e.write(fmt.Sprint(value)) // TODO: If value does not implement fmt.Stringer, what then ?
	}

	if err := e.resolveAnnotation(value, expr.Annotation); err != nil {
		return fmt.Errorf("resolve annotation: %w", err)
	}

	return nil
}

// resolveValue resolves the value of an expression's operand.
//
//   - If the operand is a literal, it returns the literal's value.
//   - If the operand is a variable, it returns the value of the variable from the input map.
func (e *executer) resolveValue(v ast.Value) (any, error) {
	switch v := v.(type) {
	default:
		return nil, fmt.Errorf("unknown value type: '%T'", v)
	case nil:
		return v, nil // nil is also a valid value.
	case ast.QuotedLiteral:
		return string(v), nil
	case ast.NameLiteral:
		return string(v), nil
	case ast.NumberLiteral:
		return float64(v), nil
	case ast.Variable:
		val, ok := e.input[string(v)]
		if !ok {
			return nil, fmt.Errorf("%w '%s'", ErrUnresolvedVariable, v)
		}

		return val, nil
	}
}

func (e *executer) resolveAnnotation(operand any, annotation ast.Annotation) error {
	annoFn, ok := annotation.(ast.Function)
	if !ok {
		return fmt.Errorf("%w with %T annotation: '%s'", ErrUnsupportedExpression, annotation, annotation)
	}

	fn, ok := e.template.funcs[annoFn.Identifier.Name]
	if !ok {
		return fmt.Errorf("%w '%s'", ErrUnknownFunction, annoFn.Identifier.Name)
	}

	opts, err := e.resolveOptions(annoFn.Options)
	if err != nil {
		return fmt.Errorf("resolve options: %w", err)
	}

	result, err := fn(operand, opts)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFormatting, err.Error())
	}

	return e.write(result)
}

func (e *executer) resolveOptions(options []ast.Option) (map[string]any, error) {
	m := make(map[string]any, len(options))

	for _, opt := range options {
		name := opt.Identifier.Name
		if _, ok := m[name]; ok {
			return nil, fmt.Errorf("%w '%s'", ErrDuplicateOptionName, name)
		}

		value, err := e.resolveValue(opt.Value)
		if err != nil {
			return nil, fmt.Errorf("resolve value: %w", err)
		}

		m[name] = value
	}

	return m, nil
}
