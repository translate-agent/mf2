package template

import (
	"errors"
	"fmt"
	"io"
	"strings"

	ast "go.expect.digital/mf2/parse"
	"go.expect.digital/mf2/template/registry"
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
	ErrUnsupportedStatement  = errors.New("unsupported statement")
	ErrDuplicateDeclaration  = errors.New("duplicate declaration")
)

// Func is a function, that will be called when a function is encountered in the template.
type Func func(operand any, options map[string]any) (string, error)

// Template represents a MessageFormat2 template.
type Template struct {
	// TODO: locale field. Can change the output of some functions.
	// e.g. number formatting, given example { $num :number }:
	//  - "en-US" -> 1,234.56
	//  - "lv-LV" -> 1234,56
	// e.g. date formatting, given example { $date :datetime }:
	//  - "en-US" -> 1/2/2023
	//  - "lv-LV" -> 2.1.2023
	ast          *ast.AST
	funcRegistry registry.Registry
}

// AddFunc adds a function to the template's function map.
func (t *Template) AddFunc(f registry.Func) {
	t.funcRegistry[f.Name] = f
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
func New() *Template {
	return &Template{funcRegistry: registry.NewRegistry()}
}

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
		for _, pattern := range message {
			if err := e.resolvePattern(pattern); err != nil {
				return fmt.Errorf("resolve pattern: %w", err)
			}
		}
	case ast.ComplexMessage:
		return e.resolveComplexMessage(message)
	}

	return nil
}

func (e *executer) resolveComplexMessage(message ast.ComplexMessage) error {
	if err := e.resolveDeclarations(message.Declarations); err != nil {
		return fmt.Errorf("resolve declarations: %w", err)
	}

	if err := e.resolveComplexBody(message.ComplexBody); err != nil {
		return fmt.Errorf("resolve complex body: %w", err)
	}

	return nil
}

func (e *executer) resolveDeclarations(declarations []ast.Declaration) error {
	m := make(map[ast.Value]struct{}, len(declarations))

	for _, decl := range declarations {
		switch d := decl.(type) {
		case ast.ReservedStatement:
			return fmt.Errorf("%w: '%s'", ErrUnsupportedStatement, "reserved statement")
		case ast.LocalDeclaration:
			if _, ok := m[d.Variable]; ok {
				return fmt.Errorf("%w '%s'", ErrDuplicateDeclaration, d)
			}

			m[d.Variable] = struct{}{}

			resolved, err := e.resolveExpression(d.Expression)
			if err != nil {
				return fmt.Errorf("resolve expression: %w", err)
			}

			e.input[string(d.Variable)] = resolved

		case ast.InputDeclaration:
			if _, ok := m[d.Operand]; ok {
				return fmt.Errorf("%w '%s'", ErrDuplicateDeclaration, d)
			}

			m[d.Operand] = struct{}{}

			resolved, err := e.resolveExpression(ast.Expression(d))
			if err != nil {
				return fmt.Errorf("resolve expression: %w", err)
			}

			e.input[string(d.Operand.(ast.Variable))] = resolved //nolint: forcetypeassert // Will always be a variable.
		}
	}

	return nil
}

func (e *executer) resolveComplexBody(body ast.ComplexBody) error {
	switch b := body.(type) {
	case ast.Matcher:
		return errors.New("matcher not implemented")
	case ast.QuotedPattern:
		for _, p := range b {
			if err := e.resolvePattern(p); err != nil {
				return fmt.Errorf("resolve pattern: %w", err)
			}
		}
	}

	return nil
}

func (e *executer) resolvePattern(pattern ast.Pattern) error {
	switch patternType := pattern.(type) {
	case ast.TextPattern:
		if err := e.write(string(patternType)); err != nil {
			return fmt.Errorf("write text pattern: %w", err)
		}
	case ast.Expression:
		resolved, err := e.resolveExpression(patternType)
		if err != nil {
			return fmt.Errorf("resolve expression: %w", err)
		}

		if err := e.write(resolved); err != nil {
			return fmt.Errorf("resolve expression: %w", err)
		}
	case ast.Markup:
		return errors.New("matcher not implemented")
	}

	return nil
}

func (e *executer) resolveExpression(expr ast.Expression) (string, error) {
	value, err := e.resolveValue(expr.Operand)
	if err != nil {
		return "", fmt.Errorf("resolve value: %w", err)
	}

	if expr.Annotation == nil {
		// NOTE: Parser won't allow value to be nil if annotation is nil.
		return fmt.Sprint(value), nil // TODO: If value does not implement fmt.Stringer, what then ?
	}

	resolved, err := e.resolveAnnotation(value, expr.Annotation)
	if err != nil {
		return "", fmt.Errorf("resolve annotation: %w", err)
	}

	return resolved, nil
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

func (e *executer) resolveAnnotation(operand any, annotation ast.Annotation) (string, error) {
	annoFn, ok := annotation.(ast.Function)
	if !ok {
		return "", fmt.Errorf("%w with %T annotation: '%s'", ErrUnsupportedExpression, annotation, annotation)
	}

	registryF, ok := e.template.funcRegistry[annoFn.Identifier.Name]
	if !ok {
		return "", fmt.Errorf("%w '%s'", ErrUnknownFunction, annoFn.Identifier.Name)
	}

	opts, err := e.resolveOptions(annoFn.Options)
	if err != nil {
		return "", fmt.Errorf("resolve options: %w", err)
	}

	result, err := registryF.Format(operand, opts)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrFormatting, err.Error())
	}

	return fmt.Sprint(result), nil
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
