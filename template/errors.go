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
	switch annotation.(type) {
	default:
		return ErrUnsupportedExpression
	case ast.PrivateUseAnnotation:
		return fmt.Errorf("%w with private use annotation: '%s'", ErrUnsupportedExpression, annotation)
	case ast.ReservedAnnotation:
		return fmt.Errorf("%w with reserved annotation: '%s'", ErrUnsupportedExpression, annotation)
	}
}
