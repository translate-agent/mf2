package mf2

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	ast "go.expect.digital/mf2/parse"
)

type Template struct {
	// TODO: add text direction, locale (fallback locales), fallback string.
	*ast.AST
	*funcReg
	err  error
	name string
}

type funcReg struct {
	parseFuncs FuncMap
	execFuncs  map[string]reflect.Value
}

type FuncMap map[string]any

func (t *Template) Funcs(funcMap FuncMap) *Template {
	if t.err != nil {
		return t
	}

	if t.AST != nil {
		t.err = errors.New("template already parsed")
		return t
	}

	if len(funcMap) != 0 {
		t.funcReg = new(funcReg)

		if t.funcReg.execFuncs, t.err = addValueFuncs(funcMap); t.err != nil {
			t.err = fmt.Errorf("add value funcs: %w", t.err)
			return t
		}

		t.funcReg.parseFuncs = addFuncs(funcMap)
	}

	return t
}

// NewTemplate creates a new template with the given name.
func NewTemplate(name string) *Template {
	return &Template{name: name}
}

// Parse parses a MF2 string into a template.
func (t *Template) Parse(s string) (*Template, error) {
	if t.err != nil {
		return nil, t.err
	}

	if *t.AST, t.err = ast.Parse(s); t.err != nil {
		t.err = fmt.Errorf("parse MF2 string: %w", t.err)
	}

	return t, nil
}

// Parse parses a MF2 string into a template, panics on error.
func (t *Template) MustParse(s string) *Template {
	if t.err != nil {
		panic(t.err)
	}

	ast, err := ast.Parse(s)
	if err != nil {
		panic(fmt.Errorf("parse MF2 string: %w", t.err))
	}

	t.AST = &ast

	return t
}

// TODO: Thread safety.
func (t *Template) Execute(wr io.Writer, data any) error {
	if t.err != nil {
		return t.err
	}

	return t.execute(wr, data)
}

func (t *Template) execute(wr io.Writer, data any) error {
	if t.err != nil {
		return t.err
	}

	if t.AST == nil {
		return fmt.Errorf("%q is an incomplete or empty template", t.name)
	}

	switch data.(type) {
	case nil, map[string]any:
		// noop
	default:
		return fmt.Errorf("unsupported input data type: %T", data)
	}

	switch message := t.AST.Message.(type) {
	case ast.SimpleMessage:
		if err := t.resolveSimpleMessage(wr, message, data); err != nil {
			return fmt.Errorf("resolve simple message: %w", err)
		}
	case ast.ComplexMessage:
		if err := t.resolveComplexMessage(wr, message, data); err != nil {
			return fmt.Errorf("resolve complex message: %w", err)
		}
	default:
		return fmt.Errorf("unsupported message type: %T", t.AST.Message)
	}

	return nil
}

// func (operand any, options []map[string]any) (string, error)

func isValidFn(typ reflect.Type) bool {
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == reflect.TypeOf((*error)(nil)).Elem():
		return true
	}

	return false
}

func addValueFuncs(funcMap FuncMap) (map[string]reflect.Value, error) {
	m := make(map[string]reflect.Value, len(funcMap))

	for name, fn := range funcMap {
		if !isValidFnName(name) {
			return nil, fmt.Errorf("invalid function name: %q", name)
		}

		v := reflect.ValueOf(fn)
		if v.Kind() != reflect.Func {
			return nil, fmt.Errorf("value for %q is not a function", name)
		}

		if !isValidFn(v.Type()) {
			return nil, fmt.Errorf("method/function %q must return 1 or 2 values, has %d return values instead",
				name, v.Type().NumOut())
		}

		m[name] = v
	}

	return m, nil
}

func addFuncs(funcMap FuncMap) FuncMap {
	m := make(map[string]any, len(funcMap))

	for name, fn := range funcMap {
		m[name] = fn
	}

	return m
}

func findFunc(name string, tmpl *Template) (v reflect.Value, ok bool) {
	if tmpl != nil && tmpl.funcReg != nil {
		if fn := tmpl.execFuncs[name]; fn.IsValid() {
			return fn, true
		}
	}

	return reflect.Value{}, false
}

func findVariableValue(name string, data any) (any, error) {
	switch data := data.(type) { // TODO: add support for more types
	case map[string]any:
		if v, ok := data[name]; ok {
			return v, nil
		}
	default:
		return "", fmt.Errorf("unsupported input data type: %T", data)
	}

	return nil, fmt.Errorf("variable %q not found", name)
}

func (t *Template) resolveSimpleMessage(wr io.Writer, msg ast.SimpleMessage, data any) error {
	for i := range msg {
		switch pattern := msg[i].(type) {
		case ast.Markup:
			// TODO: write implementation.
		case ast.LiteralExpression:
			// TODO: write implementation.
		case ast.AnnotationExpression:
			// TODO: write implementation.
		case ast.TextPattern:
			if _, err := wr.Write([]byte(pattern)); err != nil {
				return fmt.Errorf("write text: %w", err)
			}
		case ast.VariableExpression:
			switch v := pattern.Annotation.(type) {
			case ast.PrivateUseAnnotation:
			// TODO: write implementation.
			case ast.ReservedAnnotation:
			// TODO: write implementation.
			case ast.Function:
				varVal, err := findVariableValue(pattern.Variable.String(), data)
				if err != nil {
					return fmt.Errorf("resolve variable expr: find variable value: %w", err)
				}

				fnResult, err := t.evalFnCall(v.Identifier.Name, varVal, v.Options)
				if err != nil {
					return fmt.Errorf("resolve variable expr: evaluate function call: %w", err)
				}

				if _, err := wr.Write([]byte(fmt.Sprint(fnResult))); err != nil {
					return fmt.Errorf("resolve variable expr: write result of function: %w", err)
				}
			case nil: // no function
				val, err := findVariableValue(pattern.Variable.String(), data)
				if err != nil {
					return fmt.Errorf("resolve variable expr: find variable value: %w", err)
				}

				switch v := val.(type) {
				case string:
					if _, err := wr.Write([]byte(v)); err != nil {
						return fmt.Errorf("write text: %w", err)
					}
				default:
					return fmt.Errorf("unsupported variable value type: %T", v)
				}
			}
		}
	}

	return nil
}

func (t *Template) resolveComplexMessage(wr io.Writer, msg ast.ComplexMessage, data any) error {
	// TODO: write implementation.
	return errors.New("not implemented")
}

func isValidFnName(name string) bool {
	return true
	// TODO: write implementation

	// if name == "" {
	// 	return false
	// }

	// for i, r := range name {
	// 	switch {
	// 	case r == '_':
	// 	case i == 0 && !unicode.IsLetter(r):
	// 		return false
	// 	case !unicode.IsLetter(r) && !unicode.IsDigit(r):
	// 		return false
	// 	}
	// }
}

func (t *Template) evalFnCall(fnName string, varValue any, opts []ast.Option) (any, error) {
	fn, ok := findFunc(fnName, t)
	if !ok {
		return nil, fmt.Errorf("resolve variable expr: unknown function: %q", fnName)
	}

	// TODO: ast.Option identifier name can contain many "illegal in golang" characters, find a solution.
	// TODO: ast.Option value can contain a variable, write implementation.

	args := make([]reflect.Value, 0, len(opts))
	args = append(args, reflect.ValueOf(varValue))

	for i := range opts {
		switch v := opts[i].Value.(type) {
		case ast.Variable:

			// TODO:
		case ast.QuotedLiteral:
			args = append(args, reflect.ValueOf(v.String()[1:len(v.String())-1]))
		case ast.Literal:
			args = append(args, reflect.ValueOf(opts[i].Value.String()))
		}
	}

	returnValues := fn.Call(args)

	switch v := len(returnValues); v {
	case 1:
		return returnValues[0].Interface(), nil
	case 2: //nolint:gomnd
		if returnValues[1].Interface() == nil {
			return returnValues[0].Interface(), nil
		}

		return returnValues[0].Interface(), returnValues[1].Interface().(error) //nolint:forcetypeassert
	default:
		return nil, fmt.Errorf("method/function %q must return 1 or 2 values, has %d return values instead", fnName, v)
	}
}

// NOTES:

// In a declaration the resolved value of the expression is bound to variable, which is available for use by later expressions.
// Same variable can be referenced in different ways later.

// Three types of expressions: value of local-declaration, selector, placeholder in pattern,
// also input declaration can contain variable-expression.

// An input mapping of string identifiers to values,
// defining variable values that are available during variable resolution.
// This is often determined by a user-provided argument of a formatting function call.

// How do we recognize variables in a template?
// How do we pass in variable values to a template?
// How do we pass in functions to a template?
// Will template require additional options to be passed in to alter its behavior?
// How does template resolve variants inside MF2?

// If the expression contains a private-use annotation, its resolved value is defined according to the implementation's specification.

// How are simple messages handled?
// How are complex messages handled?
// - Matcher

// type FuncMap map[string]any

// func (t *Template) Funcs(funcMap FuncMap) *Template {
// 	t.text.Funcs(template.FuncMap(funcMap))
// 	return t
// }

// // Execute applies a parsed template to the specified data object,
// // and writes the output to wr.
// // If an error occurs executing the template or writing its output,
// // execution stops, but partial results may already have been written to
// // the output writer.
// // A template may be executed safely in parallel, although if parallel
// // executions share a Writer the output may be interleaved.
// //
// // If data is a reflect.Value, the template applies to the concrete
// // value that the reflect.Value holds, as in fmt.Print.
// func (t *Template) Execute(wr io.Writer, data any) error {
// 	return t.execute(wr, data)
// }

// NOTES:
// An expression is a part of a message that will be determined during the message's formatting.
// Since a variable can be referenced in different ways later, implementations SHOULD NOT immediately fully format the value for output.

// If the expression consists of a literal, its resolved value is defined by literal resolution.
// Literal value with no annotation is always treated as a string. To represent values that are not strings as a literal, an annotation needs to be provided.
// When an expression is resolved, it MUST behave as if all preceding declarations and selectors affecting variables referenced by that expression have already been evaluated in the order in which the relevant declarations and selectors appear in the message.

// At the start of pattern selection, if the message contains any reserved statements, emit an Unsupported Statement error.
// If variable value is passed in and expression doesn't have any annotation, then value MAY be resolved according to its type.

// If a declaration exists for the variable, its resolved value is used.
// Otherwise, the variable is an implicit reference to an input value, and its value is looked up from the formatting context input mapping.

// The resolution of a variable MAY fail if no value is identified for its name.
// If this happens, an Unresolved Variable error MUST be emitted. If a variable would resolve to a fallback value, this MUST also be considered a failure.

// if function stated in the expression is not found emit - Unknown Function error and use a fallback value for the expression.
