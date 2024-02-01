package mf2

import (
	"fmt"
	"io"
	"reflect"

	ast "go.expect.digital/mf2/parse"
)

type Template struct {
	// TODO: direction of text, locale (fallback locales), fallback string.
	name string
	*ast.AST
	*funcReg
}

// function registry
type funcReg struct {
	parseFuncs FuncMap
	execFuncs  map[string]reflect.Value
}

type FuncMap map[string]any

// NewTemplate creates a new template with the given name.
func NewTemplate(name string) *Template {
	return &Template{name: name}
}

// Parse parses a MF2 string into a template.
func (t *Template) Parse(s string) (*Template, error) {
	var err error

	if *t.AST, err = ast.Parse(s); err != nil {
		return nil, fmt.Errorf("ast from MF2 string: %w", err)
	}

	return t, nil
}

// Parse parses a MF2 string into a template, panics on error.
func (t *Template) MustParse(s string) *Template {
	var err error

	if *t.AST, err = ast.Parse(s); err != nil {
		panic(fmt.Errorf("ast from MF2 string: %v", err))
	}

	return t
}

func (t *Template) Execute(wr io.Writer, data any) error {
	return t.execute(wr, data)
}

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

// TODO: Implement Markup, Matcher, thread safety.

func (t *Template) execute(wr io.Writer, data any) error {
	switch v := data.(type) {
	case map[string]any:

	default:
		return fmt.Errorf("unsupported type: %T", v)
	}

	return nil
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

// no builtins

// func builtins() FuncMap {
// 	return FuncMap{
// 		"print":   fmt.Sprint,
// 		"printf":  fmt.Sprintf,
// 		"println": fmt.Sprintln,
// 	}
// }

// 1. Create Template
// 1. Parse MF2 into AST.
// 2. Add Functions to
