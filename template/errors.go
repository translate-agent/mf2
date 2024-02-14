package template

import (
	"errors"
	"fmt"

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

func syntaxErr(err error) error {
	return fmt.Errorf("%w: %w", ErrSyntax, err)
}

func unresolvedVariableErr(v ast.Variable) error {
	return fmt.Errorf("%w '%s'", ErrUnresolvedVariable, v)
}

func unknownFunctionErr(name string) error {
	return fmt.Errorf("%w '%s'", ErrUnknownFunction, name)
}

func duplicateOptionNameErr(name string) error {
	return fmt.Errorf("%w '%s'", ErrDuplicateOptionName, name)
}

func unsupportedExpressionErr(annotation ast.Annotation) error {
	var typ string
	switch annotation.(type) {
	case ast.PrivateUseAnnotation:
		typ = "private use annotation"
	case ast.ReservedAnnotation:
		typ = "reserved annotation"
	}

	return fmt.Errorf("%w with %s: '%s'", ErrUnsupportedExpression, typ, annotation)
}

func formattingErr(err error) error {
	return fmt.Errorf("%w: %w", ErrFormatting, err)
}
