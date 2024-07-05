package mf2

import "errors"

// List of [MF2 errors] as defined in the specification.
//
// [MF2 errors]: https://github.com/unicode-org/message-format-wg/blob/main/spec/errors.md
var (
	// SYNTAX ERRORS.

	// ErrSyntax occurs when the syntax representation of a message is not well-formed.
	ErrSyntax = errors.New("syntax error")

	// DATA MODEL ERRORS.

	// ErrDuplicateDeclaration occurs when a variable is declared more than once.
	// Note that an input variable is implicitly declared when it is first used,
	// so explicitly declaring it after such use is also an error.
	ErrDuplicateDeclaration = errors.New("duplicate declaration")
	// ErrDuplicateOptionName occurs when the same identifier
	// appears on the left-hand side of more than one option in the same expression.
	ErrDuplicateOptionName = errors.New("duplicate option name")
	// ErrMissingFallbackVariant occurs when the number of keys on a variant
	// does not equal the number of selectors.
	ErrMissingFallbackVariant = errors.New("missing fallback variant")
	// ErrMissingSelectorAnnotation occurs when the message
	// contains a selector that does not have an annotation,
	// or contains a variable that does not directly or indirectly reference a declaration with an annotation.
	ErrMissingSelectorAnnotation = errors.New("missing selector annotation")
	// ErrVariantKeyMismatch occurs when the number of keys on a variant
	// does not equal the number of selectors.
	ErrVariantKeyMismatch = errors.New("variant key mismatch")

	// RESOLUTION ERRORS.

	// ErrUnknownFunction occurs when an expression includes
	// a reference to a function which cannot be resolved.
	ErrUnknownFunction = errors.New("unknown function")
	// ErrUnresolvedVariable occurs when a variable reference cannot be resolved.
	ErrUnresolvedVariable = errors.New("unresolved variable")
	// ErrUnsupportedExpression occurs when an expression uses
	// syntax reserved for future standardization,
	// or for private implementation use that is not supported by the current implementation.
	ErrUnsupportedExpression = errors.New("unsupported expression")
	// ErrUnsupportedStatement occurs when a message includes a reserved statement.
	ErrUnsupportedStatement = errors.New("unsupported statement")

	// MESSAGE FUNCTION ERRORS.

	// ErrBadOperand is any error that occurs due to the content or format of the operand,
	// such as when the operand provided to a function during function resolution does not match one of the
	// expected implementation-defined types for that function;
	// or in which a literal operand value does not have the required format
	// and thus cannot be processed into one of the expected implementation-defined types
	// for that specific function.
	ErrBadOperand = errors.New("bad operand")
	// ErrBadOption is an error that occurs when there is
	// an implementation-defined error with an option or its value.
	ErrBadOption = errors.New("bad option")
	// ErrBadVariantKey is an error that occurs when a variant key
	// does not match the expected implementation-defined format.
	ErrBadVariantKey = errors.New("bad variant key")
)
