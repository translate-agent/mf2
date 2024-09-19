package mf2

import (
	"errors"
	"fmt"
)

// List of [MF2 errors] as defined in the specification.
//
// [MF2 errors]: https://github.com/unicode-org/message-format-wg/blob/main/spec/errors.md
var (
	// SYNTAX ERRORS.

	// ErrSyntax occurs when the syntax representation of a message is not well-formed.
	ErrSyntax = errors.New("syntax error")

	// DATA MODEL ERRORS.

	// ErrDataModel occur when a message is not valid due to
	// violating one of the semantic requirements on its structure.
	//
	// ErrDataModel is returned with:
	//	- ErrDuplicateDeclaration
	//	- ErrDuplicateOptionName
	//	- ErrDuplicateVariant
	//	- ErrMissingFallbackVariant
	//	- ErrMissingSelectorAnnotation
	//	- ErrVariantKeyMismatch
	ErrDataModel = errors.New("data model error")

	// ErrDuplicateDeclaration occurs when a variable is declared more than once.
	// Note that an input variable is implicitly declared when it is first used,
	// so explicitly declaring it after such use is also an error.
	ErrDuplicateDeclaration = fmt.Errorf("%w: %w", ErrDataModel, errors.New("duplicate declaration"))
	// ErrDuplicateOptionName occurs when the same identifier
	// appears on the left-hand side of more than one option in the same expression.
	ErrDuplicateOptionName = fmt.Errorf("%w: %w", ErrDataModel, errors.New("duplicate option name"))
	// ErrDuplicateVariant error occurs when the same list of keys is used
	// for more than one variant.
	ErrDuplicateVariant = fmt.Errorf("%w: %w", ErrDataModel, errors.New("duplicate variant"))
	// ErrMissingFallbackVariant occurs when the number of keys on a variant
	// does not equal the number of selectors.
	ErrMissingFallbackVariant = fmt.Errorf("%w: %w", ErrDataModel, errors.New("missing fallback variant"))
	// ErrMissingSelectorAnnotation occurs when the message
	// contains a selector that does not have an annotation,
	// or contains a variable that does not directly or indirectly reference a declaration with an annotation.
	ErrMissingSelectorAnnotation = fmt.Errorf("%w: %w", ErrDataModel, errors.New("missing selector annotation"))
	// ErrVariantKeyMismatch occurs when the number of keys on a variant
	// does not equal the number of selectors.
	ErrVariantKeyMismatch = fmt.Errorf("%w: %w", ErrDataModel, errors.New("variant key mismatch"))

	// RESOLUTION ERRORS.

	// ErrResolution occur when the runtime value of a part of a message
	// cannot be determined.
	//
	// ErrResolution is returned with:
	//	- ErrUnknownFunction
	//	- ErrUnresolvedVariable
	//	- ErrUnsupportedExpression
	//	- ErrUnsupportedStatement
	ErrResolution = errors.New("resolution error")

	// ErrUnknownFunction occurs when an expression includes
	// a reference to a function which cannot be resolved.
	ErrUnknownFunction = fmt.Errorf("%w: %w", ErrResolution, errors.New("unknown function"))
	// ErrUnresolvedVariable occurs when a variable reference cannot be resolved.
	ErrUnresolvedVariable = fmt.Errorf("%w: %w", ErrResolution, errors.New("unresolved variable"))
	// ErrUnsupportedExpression occurs when an expression uses
	// syntax reserved for future standardization,
	// or for private implementation use that is not supported by the current implementation.
	ErrUnsupportedExpression = fmt.Errorf("%w: %w", ErrResolution, errors.New("unsupported expression"))
	// ErrUnsupportedStatement occurs when a message includes a reserved statement.
	ErrUnsupportedStatement = fmt.Errorf("%w: %w", ErrResolution, errors.New("unsupported statement"))

	// MESSAGE FUNCTION ERRORS.

	// ErrMessageFunction is any error that occurs when calling a message function implementation
	// or which depends on validation associated with a specific function.
	//
	// ErrMessageFunction is returned with:
	//	- ErrBadOperand
	//	- ErrBadOption
	//	- ErrBadSelector
	//	- ErrBadVariantKey
	ErrMessageFunction = errors.New("message function error")

	// ErrBadOperand is any error that occurs due to the content or format of the operand,
	// such as when the operand provided to a function during function resolution does not match one of the
	// expected implementation-defined types for that function;
	// or in which a literal operand value does not have the required format
	// and thus cannot be processed into one of the expected implementation-defined types
	// for that specific function.
	ErrBadOperand = fmt.Errorf("%w: %w", ErrMessageFunction, errors.New("bad operand"))
	// ErrBadOption is an error that occurs when there is
	// an implementation-defined error with an option or its value.
	ErrBadOption = fmt.Errorf("%w: %w", ErrMessageFunction, errors.New("bad option"))
	// ErrBadSelector error occurs when a message includes a selector
	// with a resolved value which does not support selection.
	ErrBadSelector = fmt.Errorf("%w: %w", ErrMessageFunction, errors.New("bad selector"))
	// ErrBadVariantKey is an error that occurs when a variant key
	// does not match the expected implementation-defined format.
	ErrBadVariantKey = fmt.Errorf("%w: %w", ErrMessageFunction, errors.New("bad variant key"))
)
