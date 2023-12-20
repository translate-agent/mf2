package mf2

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

// Message is the top-level node.
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

// ---------------------------------Types------------------------------------
//
// Here we define the types that implement the interfaces defined above.
//
// Types with one concrete field (string, int, ...) are defined as types
// Types with one interface field are defined as structs
// Types with multiple fields are defined as structs

// ---------------------------------Message------------------------------------

type SimpleMessage struct {
	Message

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

type ComplexMessage struct {
	Message

	ComplexBody  ComplexBody   // Matcher or QuotedPattern
	Declarations []Declaration // Optional: InputDeclaration, LocalDeclaration or ReservedStatement
}

// ---------------------------------Pattern------------------------------------

type TextPattern string

func (TextPattern) node()    {}
func (TextPattern) pattern() {}

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

	Annotation Annotation // Optional: FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
	Variable   Variable
}

type AnnotationExpression struct {
	Expression

	Annotation Annotation // FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

// ---------------------------------Literal------------------------------------

type QuotedLiteral string

func (QuotedLiteral) node()    {}
func (QuotedLiteral) literal() {}

type UnquotedLiteral struct {
	Literal

	Value Unquoted // NameLiteral or NumberLiteral
}

type NameLiteral string

func (NameLiteral) node()     {}
func (NameLiteral) literal()  {}
func (NameLiteral) unquoted() {}

type NumberLiteral float64

func (NumberLiteral) node()     {}
func (NumberLiteral) literal()  {}
func (NumberLiteral) unquoted() {}

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

	Literal    Literal // QuotedLiteral or UnquotedLiteral
	Identifier Identifier
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

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
	Variable   Variable
}

type ReservedStatement struct {
	Declaration

	// TODO: Implementation
}

// --------------------------------VariantKey----------------------------------

type LiteralKey struct {
	VariantKey

	Literal Literal // QuotedLiteral or UnquotedLiteral
}

// CatchAllKey is a special key, that matches any value.
type CatchAllKey struct {
	VariantKey
}

// ---------------------------------ComplexBody--------------------------------------

type QuotedPattern struct {
	ComplexBody

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

type Matcher struct {
	ComplexBody

	MatchStatements []Expression // At least one
	Variants        []Variant    // At least one
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

	Identifier Identifier
	Prefix     rune // One of: ':', '+', '-'
}

type Variant struct {
	Node

	Keys          []VariantKey // At least one: LiteralKey or WildcardKey
	QuotedPattern QuotedPattern
}
