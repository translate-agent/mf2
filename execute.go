package mf2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	ast "go.expect.digital/mf2/parse"
)

// TODO: add text direction, locale (fallback locales), fallback string, thread safety.

type Template struct {
	*ast.AST
	*funcReg
	err  error
	name string
}

type funcReg struct {
	parseFuncs FuncMap
	execFuncs  map[string]reflect.Value
}

// FuncMap is a map of functions that can be added to a template.
type FuncMap map[string]func(operator any, options map[string]any) (any, error)

// Funcs adds the functions to the template's function map.
func (t *Template) Funcs(funcMap FuncMap) *Template {
	if t.err != nil {
		return t
	}

	if t.AST != nil {
		t.err = errors.New("functions must be added before Parse or MustParse method is called")
		return t
	}

	if len(funcMap) != 0 {
		t.funcReg = new(funcReg)
		t.funcReg.execFuncs = make(map[string]reflect.Value, len(funcMap))
		t.funcReg.parseFuncs = make(FuncMap, len(funcMap))

		for name, fn := range funcMap { // TODO: validate function name
			t.funcReg.execFuncs[name] = reflect.ValueOf(fn)
		}

		for name, fn := range funcMap {
			t.funcReg.parseFuncs[name] = fn
		}
	}

	return t
}

// NewTemplate creates a new template with the given name.
func NewTemplate(name string) *Template {
	return &Template{name: name}
}

// Parse parses a MF2 string into a template, returns an error if parsing fails.
func (t *Template) Parse(s string) (*Template, error) {
	if t.err != nil {
		return nil, t.err
	}

	if *t.AST, t.err = ast.Parse(s); t.err != nil {
		return nil, fmt.Errorf("parse MF2 string: %w", t.err)
	}

	return t, nil
}

// MustParse parses a MF2 string into the template, panics on error.
func (t *Template) MustParse(s string) *Template {
	if t.err != nil {
		panic(t.err)
	}

	var err error

	if *t.AST, err = ast.Parse(s); err != nil {
		panic(fmt.Errorf("parse MF2 string: %w", t.err))
	}

	return t
}

// Execute applies a parsed template to the specified input data object,
// and writes the output to io.Writer.
// Currently only supports input data of type map[string]any.
func (t *Template) Execute(wr io.Writer, input any) error {
	if _, ok := input.(map[string]any); !ok {
		return fmt.Errorf("unsupported input data type %T, must be of map[string]any", input)
	}

	if t.err != nil {
		return fmt.Errorf("execute template %q: %w", t.name, t.err)
	}

	return t.execute(wr, input)
}

func (t *Template) execute(wr io.Writer, input any) error {
	if t.err != nil {
		return t.err
	}

	if t.AST == nil {
		return fmt.Errorf("no parsed MF2 message found")
	}

	switch message := t.AST.Message.(type) {
	case ast.SimpleMessage:
		if err := t.resolveSimpleMessage(wr, message, input); err != nil {
			return fmt.Errorf("resolve simple message: %w", err)
		}
	case ast.ComplexMessage:
		if err := t.resolveComplexMessage(wr, message, input); err != nil {
			return fmt.Errorf("resolve complex message: %w", err)
		}
	default:
		return fmt.Errorf("unsupported MF2 message type: %T", t.AST.Message)
	}

	return nil
}

// resolveSimpleMessage resolves a MF2 simple message and writes the output to io.Writer.
func (t *Template) resolveSimpleMessage(wr io.Writer, sm ast.SimpleMessage, input any) error {
	for i := range sm {
		switch pattern := sm[i].(type) {
		case ast.Expression:
			if err := t.resolveExpr(wr, pattern, input); err != nil {
				return fmt.Errorf("resolve expression: %w", err)
			}
		case ast.TextPattern:
			if _, err := wr.Write([]byte(pattern)); err != nil {
				return fmt.Errorf("write text: %w", err)
			}
		case ast.Markup:
			if err := t.resolveMarkup(wr, pattern, input); err != nil {
				return fmt.Errorf("resolve markup: %w", err)
			}
		default:
			return fmt.Errorf("unsupported pattern type: %T", pattern)
		}
	}

	return nil
}

// resolveComplexMessage resolves a MF2 complex message and writes the result to io.Writer.
func (t *Template) resolveComplexMessage(wr io.Writer, cm ast.ComplexMessage, input any) error {
	return errors.New("not implemented") // TODO: write implementation.
}

func (t *Template) resolveExpr(wr io.Writer, expr ast.Expression, input any) error {
	switch annotation := expr.Annotation.(type) {
	case nil, ast.Function, ast.PrivateUseAnnotation, ast.ReservedAnnotation:
		switch expr.Operand.(type) {
		case ast.Literal:
			if err := t.resolveLiteralExpr(wr, expr, input); err != nil {
				return fmt.Errorf("resolve literal expr: %w", err)
			}
		case ast.Variable:
			if err := t.resolveVariableExpr(wr, expr, input); err != nil {
				return fmt.Errorf("resolve variable expr: %w", err)
			}
		}
	default:
		return fmt.Errorf("unsupported annotation type: %T", annotation)
	}

	return nil
}

func (t *Template) resolveLiteralExpr(wr io.Writer, expr ast.Expression, input any) error {
	literal, ok := expr.Operand.(ast.Literal)
	if !ok {
		return fmt.Errorf("invalid operand type: %T", expr.Operand)
	}

	var result string

	switch v := expr.Annotation.(type) {
	case nil:
		result = literal.String()
	case ast.Function:
		fnResult, err := t.evalFnCall(v, input)
		if err != nil {
			return fmt.Errorf("evaluate function call: %w", err)
		}

		if result, err = toString(fnResult); err != nil {
			return fmt.Errorf("function result to string: %w", err)
		}
	case ast.PrivateUseAnnotation:
		return fmt.Errorf("private-use-annotation not implemented") // TODO: write implementation.
	case ast.ReservedAnnotation:
		return fmt.Errorf("reserved-annotation not implemented") // TODO: write implementation.
	}

	if _, err := wr.Write([]byte(result)); err != nil {
		return fmt.Errorf("write variable value: %w", err)
	}

	return nil
}

func (t *Template) resolveVariableExpr(wr io.Writer, expr ast.Expression, input any) error {
	variable, ok := expr.Operand.(ast.Variable)
	if !ok {
		return fmt.Errorf("invalid operand type: %T", expr.Operand)
	}

	value, err := findVarValue(variable.String(), input)
	if err != nil {
		return fmt.Errorf("find variable value: %w", err)
	}

	var result string

	switch v := expr.Annotation.(type) {
	case nil:
		if result, err = toString(value); err != nil {
			return fmt.Errorf("variable value to string: %w", err)
		}
	case ast.Function:
		fnResult, err := t.evalFnCall(v, input)
		if err != nil {
			return fmt.Errorf("evaluate function call: %w", err)
		}

		if result, err = toString(fnResult); err != nil {
			return fmt.Errorf("function result to string: %w", err)
		}
	case ast.PrivateUseAnnotation:
		return fmt.Errorf("private-use-annotation not implemented") // TODO: write implementation.
	case ast.ReservedAnnotation:
		return fmt.Errorf("reserved-annotation not implemented") // TODO: write implementation.
	}

	if _, err := wr.Write([]byte(result)); err != nil {
		return fmt.Errorf("write variable value: %w", err)
	}

	return nil
}

func (t *Template) resolveMarkup(wr io.Writer, markup ast.Markup, input any) error {
	return errors.New("not implemented") // TODO: write implementation.
}

// findFunc finds a function with the given name in the template's function map.
func (t *Template) findFunc(name string) (reflect.Value, error) {
	if t.funcReg != nil {
		if fn, ok := t.execFuncs[name]; ok {
			return fn, nil
		}
	}

	return reflect.Value{}, fmt.Errorf("function %q not found in function registry", name)
}

// findVarValue finds the value of a variable in the input data.
func findVarValue(name string, input any) (any, error) {
	switch v := input.(type) {
	case map[string]any:
		if val, ok := v[name]; ok {
			return val, nil
		}
	default:
		return nil, fmt.Errorf("unsupported input data type %T, must be of map[string]any", input)
	}

	return nil, fmt.Errorf("variable %q not found", name)
}

// evalFnCall looks for a function with the given name in the template's function map and calls it, returns the result.
func (t *Template) evalFnCall(astFn ast.Function, input any, args ...any) (any, error) {
	fn, err := t.findFunc(astFn.Identifier.Name)
	if err != nil {
		return nil, fmt.Errorf("find function: %w", err)
	}

	fnArgs := make([]reflect.Value, 0, len(args))

	for i := range args {
		fnArgs = append(fnArgs, reflect.ValueOf(args[i]))
	}

	for i := range astFn.Options {
		switch v := astFn.Options[i].Value.(type) {
		case ast.Variable:
			value, err := findVarValue(v.String(), input)
			if err != nil {
				return nil, fmt.Errorf("find variable value: %w", err)
			}

			fnArgs = append(fnArgs, reflect.ValueOf(value))
		case ast.QuotedLiteral:
			args = append(args, reflect.ValueOf(v.String()[1:len(v.String())-1]))
		case ast.Literal:
			args = append(args, reflect.ValueOf(v.String()))
		}
	}

	fnResult := fn.Call(fnArgs)

	switch len(fnResult) {
	case 1:
		return fnResult[0].Interface(), nil
	case 2: //nolint:gomnd
		switch v := fnResult[1].Interface().(type) {
		case nil:
			return fnResult[0].Interface(), nil
		case error:
			return nil, v
		default:
			return nil, fmt.Errorf("function %q returned %T as second value, expected error", astFn.Identifier.Name, v)
		}
	}

	return nil, fmt.Errorf("function %q returned %d values, expected 1 or 2", astFn.Identifier.Name, len(fnResult))
}

// toString converts input value to string.
func toString(v any) (string, error) {
	var s string

	switch v := v.(type) {
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("marshal variable value: %w", err)
		}

		s = string(b)
	case string:
		return v, nil
	}

	return s, nil
}
