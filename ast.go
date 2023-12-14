package mf2

import (
	"golang.org/x/exp/constraints"
)

// AST is the abstract syntax tree of a MessageFormat 2.0 source file.
type AST Message

// --------------------------------Interfaces----------------------------------
//
// Here we define the Nodes that can have multiple types.
// For example Message could be either a SimpleMessage or a ComplexMessage.
// Pattern could be either a TextPattern or a PlaceholderPattern.
// etc.

// Node is the interface implemented by all AST nodes.
type Node interface {
	node()
}

// Message is the top-level node
type Message interface {
	Node
	message()
}

type Pattern interface {
	Node
	pattern()
}

type Expression interface {
	Node
	expression()
}

type Literal interface {
	Node
	literal()
}

type Unquoted interface {
	Literal
	unquoted()
}

type Annotation interface {
	Node
	annotation()
}

type Option interface {
	Node
	option()
}

type Declaration interface {
	Node
	declaration()
}

type ComplexBody interface {
	Node
	complexBody()
}

type VariantKey interface {
	Node
	variantKey()
}

// ---------------------------------Structs------------------------------------
//
// Here we define the structs that implement the interfaces defined above.

// ---------------------------------Message------------------------------------

type SimpleMessage struct {
	Message

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

type ComplexMessage struct {
	Message

	Declarations []Declaration // Optional: InputDeclaration, LocalDeclaration or ReservedStatement
	ComplexBody  ComplexBody   // Matcher or QuotedPattern
}

// ---------------------------------Pattern------------------------------------

type TextPattern struct {
	Pattern

	Text string
}

type PlaceholderPattern struct {
	Pattern

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
}

// --------------------------------Expression----------------------------------

type LiteralExpression struct {
	Expression

	Literal    Literal    // QuotedLiteral or UnquotedLiteral
	Annotation Annotation // Optional: FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

type VariableExpression struct {
	Expression

	Variable   Variable
	Annotation Annotation // Optional: FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

type AnnotationExpression struct {
	Expression

	Annotation Annotation // FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

// ---------------------------------Literal------------------------------------

type QuotedLiteral struct {
	Literal

	Value string
}

type UnquotedLiteral struct {
	Literal

	Value Unquoted // NameLiteral or NumberLiteral
}

type NameLiteral struct {
	Unquoted

	Name string
}

type NumberLiteral[T constraints.Integer | constraints.Float] struct {
	Unquoted

	Number T
}

// --------------------------------Annotation----------------------------------

type FunctionAnnotation struct {
	Annotation

	Function Function
	Options  []Option // Optional: LiteralOption or VariableOption
}

type PrivateUseAnnotation struct {
	Annotation

	// TODO: Implementation
}

type ReservedAnnotation struct {
	Annotation

	// TODO: Implementation
}

// ---------------------------------Option-------------------------------------

type LiteralOption struct {
	Option

	Identifier Identifier
	Literal    Literal // QuotedLiteral or UnquotedLiteral
}

type VariableOption struct {
	Option

	Identifier Identifier
	Variable   Variable
}

// --------------------------------Declaration---------------------------------

type InputDeclaration struct {
	Declaration

	Expression VariableExpression
}

type LocalDeclaration struct {
	Declaration

	Variable   Variable
	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
}

type ReservedStatement struct {
	Declaration

	// todo: Implementation
}

// --------------------------------VariantKey----------------------------------

type LiteralKey struct {
	VariantKey

	Literal Literal // QuotedLiteral or UnquotedLiteral
}

type WildcardKey struct {
	VariantKey

	Wildcard rune // '*'
}

// ---------------------------------ComplexBody--------------------------------------

type QuotedPattern struct {
	ComplexBody

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

type Matcher struct {
	ComplexBody

	MatchStatement MatchStatement
	Variants       []Variant // At least one
}

// ---------------------------------Node---------------------------------

type Variable string

func (Variable) node() {}

type Identifier struct {
	Node

	Namespace string // Optional
	Name      string
}

type Function struct {
	Node

	Prefix     rune // One of: ':', '+', '-'
	Identifier Identifier
}

type MatchStatement struct {
	Node

	Selectors []Selector // At least one
}

type Selector Expression

type Variant struct {
	Node

	Key           VariantKey // At least one: LiteralKey or WildcardKey
	QuotedPattern QuotedPattern
}
