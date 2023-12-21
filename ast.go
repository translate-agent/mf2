package mf2

import (
	"fmt"
	"strings"
)

// AST is the abstract syntax tree of a MessageFormat 2.0 message.
type AST struct {
	Message Message
}

/*
String returns the string representation of the AST, i.e. MF2 formatted message.

Example:

	ast := AST{
		Message: SimpleMessage{
			Patterns: []Pattern{
				TextPattern("Hello, "),
				PlaceholderPattern{Expression: VariableExpression{Variable: Variable("variable")}},
				TextPattern(" World!"),
			},
		},
	}

	fmt.Print(ast) // Hello, { $variable } World!
*/
func (a AST) String() string { return fmt.Sprint(a.Message) }

// --------------------------------Interfaces----------------------------------
//
// Here we define the Nodes that can have multiple types.
// For example Message could be either a SimpleMessage or a ComplexMessage.
// Pattern could be either a TextPattern or a PlaceholderPattern.
// etc.

// Node is the interface implemented by all AST nodes.
type Node interface {
	node()

	fmt.Stringer
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

func (sm SimpleMessage) String() string { return sliceToString(sm.Patterns, "") }

type ComplexMessage struct {
	Message

	ComplexBody  ComplexBody   // Matcher or QuotedPattern
	Declarations []Declaration // Optional: InputDeclaration, LocalDeclaration or ReservedStatement
}

func (cm ComplexMessage) String() string {
	if len(cm.Declarations) == 0 {
		return fmt.Sprint(cm.ComplexBody)
	}

	return fmt.Sprintf("%s\n%s", sliceToString(cm.Declarations, "\n"), cm.ComplexBody)
}

// ---------------------------------Pattern------------------------------------

type TextPattern string

func (TextPattern) node()    {}
func (TextPattern) pattern() {}
func (tp TextPattern) String() string {
	// text-escape = backslash ( backslash / "{" / "}" )
	r := strings.NewReplacer(
		`\`, `\\`,
		`{`, `\{`,
		`}`, `\}`,
	)

	return r.Replace(string(tp))
}

type PlaceholderPattern struct {
	Pattern

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
}

func (pp PlaceholderPattern) String() string { return fmt.Sprint(pp.Expression) }

// --------------------------------Expression----------------------------------

type LiteralExpression struct {
	Expression

	Literal    Literal    // QuotedLiteral or UnquotedLiteral
	Annotation Annotation // Optional: FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

func (le LiteralExpression) String() string {
	if le.Annotation == nil {
		return fmt.Sprintf("{ %s }", le.Literal)
	}

	return fmt.Sprintf("{ %s %s }", le.Literal, le.Annotation)
}

type VariableExpression struct {
	Expression

	Annotation Annotation // Optional: FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
	Variable   Variable
}

func (ve VariableExpression) String() string {
	if ve.Annotation == nil {
		return fmt.Sprintf("{ %s }", ve.Variable)
	}

	return fmt.Sprintf("{ %s %s }", ve.Variable, ve.Annotation)
}

type AnnotationExpression struct {
	Expression

	Annotation Annotation // FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

func (ae AnnotationExpression) String() string { return fmt.Sprintf("{ %s }", ae.Annotation) }

// ---------------------------------Literal------------------------------------

type QuotedLiteral string

func (QuotedLiteral) node()    {}
func (QuotedLiteral) literal() {}
func (ql QuotedLiteral) String() string {
	// quoted-escape = backslash ( backslash / "|" )
	r := strings.NewReplacer(
		`\`, `\\`,
		`|`, `\|`,
	)

	return fmt.Sprintf("|%s|", r.Replace(string(ql)))
}

type UnquotedLiteral struct {
	Literal

	Value Unquoted // NameLiteral or NumberLiteral
}

func (ul UnquotedLiteral) String() string { return fmt.Sprint(ul.Value) }

type NameLiteral string

func (NameLiteral) node()             {}
func (NameLiteral) literal()          {}
func (NameLiteral) unquoted()         {}
func (nl NameLiteral) String() string { return string(nl) }

type NumberLiteral float64

func (NumberLiteral) node()             {}
func (NumberLiteral) literal()          {}
func (NumberLiteral) unquoted()         {}
func (nl NumberLiteral) String() string { return fmt.Sprint(float64(nl)) }

// --------------------------------Annotation----------------------------------

type FunctionAnnotation struct {
	Annotation

	Function Function
	Options  []Option // Optional: LiteralOption or VariableOption
}

func (fa FunctionAnnotation) String() string {
	if len(fa.Options) == 0 {
		return fmt.Sprint(fa.Function)
	}

	return fmt.Sprintf("%s %s", fa.Function, sliceToString(fa.Options, " "))
}

type PrivateUseAnnotation struct {
	Annotation

	// TODO: Implementation
}

func (PrivateUseAnnotation) String() string { return "^ PRIVATE USE ANNOTATION NOT IMPLEMENTED" } // TODO: Implement

type ReservedAnnotation struct {
	Annotation

	// TODO: Implementation
}

func (ReservedAnnotation) String() string { return "! RESERVED ANNOTATION NOT IMPLEMENTED" } // TODO: Implement

// ---------------------------------Option-------------------------------------

type LiteralOption struct {
	Option

	Literal    Literal // QuotedLiteral or UnquotedLiteral
	Identifier Identifier
}

func (lo LiteralOption) String() string { return fmt.Sprintf("%s = %s", lo.Identifier, lo.Literal) }

type VariableOption struct {
	Option

	Identifier Identifier
	Variable   Variable
}

func (vo VariableOption) String() string { return fmt.Sprintf("%s = %s", vo.Identifier, vo.Variable) }

// --------------------------------Declaration---------------------------------

type InputDeclaration struct {
	Declaration

	Expression VariableExpression
}

func (id InputDeclaration) String() string { return fmt.Sprintf("%s %s", input, id.Expression) }

type LocalDeclaration struct {
	Declaration

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
	Variable   Variable
}

func (ld LocalDeclaration) String() string {
	return fmt.Sprintf("%s %s = %s", local, ld.Variable, ld.Expression)
}

type ReservedStatement struct {
	Declaration

	// TODO: Implementation
}

func (ReservedStatement) String() string { return "TODO" }

// --------------------------------VariantKey----------------------------------

type LiteralKey struct {
	VariantKey

	Literal Literal // QuotedLiteral or UnquotedLiteral
}

func (lk LiteralKey) String() string { return fmt.Sprint(lk.Literal) }

// CatchAllKey is a special key, that matches any value.
type CatchAllKey struct {
	VariantKey
}

func (ck CatchAllKey) String() string { return catchAllSymbol }

// ---------------------------------ComplexBody--------------------------------------

type QuotedPattern struct {
	ComplexBody

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

func (qp QuotedPattern) String() string {
	return fmt.Sprintf("{{%s}}", sliceToString(qp.Patterns, ""))
}

type Matcher struct {
	ComplexBody

	MatchStatements []Expression // At least one
	Variants        []Variant    // At least one
}

func (m Matcher) String() string {
	matchStr := sliceToString(m.MatchStatements, " ")
	variantsStr := sliceToString(m.Variants, "\n")

	return fmt.Sprintf("%s %s\n%s", match, matchStr, variantsStr)
}

// ---------------------------------Node---------------------------------

type Variable string

func (Variable) node()            {}
func (v Variable) String() string { return fmt.Sprintf("%c%s", variablePrefix, string(v)) }

type Identifier struct {
	Node

	Namespace string // Optional
	Name      string
}

func (i Identifier) String() string {
	if i.Namespace == "" {
		return i.Name
	}

	return fmt.Sprintf("%s:%s", i.Namespace, i.Name)
}

type Function struct {
	Node

	Identifier Identifier
	Prefix     rune // One of: ':', '+', '-'
}

func (f Function) String() string { return fmt.Sprintf("%c%s", f.Prefix, f.Identifier) }

type Variant struct {
	Node

	Keys          []VariantKey // At least one: LiteralKey or WildcardKey
	QuotedPattern QuotedPattern
}

func (v Variant) String() string {
	return fmt.Sprintf("%s %s", sliceToString(v.Keys, " "), v.QuotedPattern)
}

// ---------------------------------Constants---------------------------------

const (
	variablePrefix = '$'
	catchAllSymbol = "*"
	match          = "." + keywordMatch
	local          = "." + keywordLocal
	input          = "." + keywordInput
)

// ---------------------------------Helpers---------------------------------

// sliceToString converts a slice of Nodes to a string, separated by sep.
func sliceToString[T Node](s []T, sep string) string {
	if len(s) == 0 {
		return ""
	}

	r := s[0].String()
	for _, v := range s[1:] {
		r += sep + v.String()
	}

	return r
}
