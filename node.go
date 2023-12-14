package mf2

import (
	"golang.org/x/exp/constraints"
)

type AST Message

// --------------------------------Interfaces----------------------------------

type Node interface {
	node()
}

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
	Node
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

type SimpleMessage struct {
	Message

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

type ComplexMessage struct {
	Message

	Declarations []Declaration // Optional: InputDeclaration or LocalDeclaration
	ComplexBody  ComplexBody   // Matcher or QuotedPattern
}

type TextPattern struct {
	Pattern

	Text string
}

type PlaceholderPattern struct {
	Pattern

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
}

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

type FunctionAnnotation struct {
	Annotation

	Function Function
	Options  []Option // Optional: LiteralOption or VariableOption
}

type PrivateUseAnnotation struct {
	Annotation

	// todo
}

type ReservedAnnotation struct {
	Annotation

	// todo
}

type Variable string

func (Variable) node() {}

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

type InputDeclaration struct {
	Declaration

	Expression VariableExpression
}

type LocalDeclaration struct {
	Declaration

	Variable   Variable
	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
}

type ReservedDeclaration struct {
	Declaration

	// todo: Implementation
}

type QuotedPattern struct {
	ComplexBody

	Patterns []Pattern
}

type Matcher struct {
	ComplexBody

	MatchStatement MatchStatement
	Variants       []Variant // At least one
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

type LiteralKey struct {
	VariantKey

	Literal Literal // QuotedLiteral or UnquotedLiteral
}

type WildcardKey struct {
	VariantKey

	Wildcard rune // '*'
}
