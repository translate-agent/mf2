package parse

import (
	"fmt"
	"strconv"
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
			Text("Hello, "),
			Expression{Operand: Variable("variable")}
			Text(" World!"),
		},
	}

	fmt.Print(ast) // Hello, { $variable } World!
*/
func (a AST) String() string { return a.Message.String() }

// --------------------------------Interfaces----------------------------------
//
// Here we define the Nodes that can have multiple types.
// For example Message could be either a SimpleMessage or a ComplexMessage.
// Pattern could be either a Text, Expression or a Markup.

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

type PatternPart interface {
	Node
	patternPart()
}

type Literal interface {
	Value
	VariantKey
	literal()
}

type Annotation interface {
	Node
	annotation()
}

// Value can be either a Literal or a Variable.
type Value interface {
	Node
	value()
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

type ReservedBody interface {
	Node
	reservedBody()
}

// ---------------------------------Types------------------------------------
//
// Here we define the types that implement the interfaces defined above.
//
// --------------------------------Message------------------------------------

type SimpleMessage []PatternPart

// String returns MF2 formatted string.
func (m SimpleMessage) String() string {
	return sliceToString(m, "")
}

func (m SimpleMessage) node()    {}
func (m SimpleMessage) message() {}

type ComplexMessage struct {
	ComplexBody  ComplexBody   // Matcher or QuotedPattern
	Declarations []Declaration // Optional: InputDeclaration, LocalDeclaration or ReservedStatement
}

// String returns MF2 formatted string.
func (m ComplexMessage) String() string {
	if len(m.Declarations) == 0 {
		return m.ComplexBody.String()
	}

	return sliceToString(m.Declarations, "\n") + "\n" + m.ComplexBody.String()
}

func (m ComplexMessage) node()    {}
func (m ComplexMessage) message() {}

// -----------------------------------Text-------------------------------------

type Text string

// String returns MF2 formatted string.
func (t Text) String() string {
	// text-escape = backslash ( backslash / "{" / "}" )
	r := strings.NewReplacer(
		`\`, `\\`,
		`{`, `\{`,
		`}`, `\}`,
	)

	return r.Replace(string(t))
}

func (Text) node()        {}
func (Text) patternPart() {}

// --------------------------------Expression----------------------------------

type Expression struct {
	Operand    Value       // Literal or Variable
	Annotation Annotation  // Function, PrivateUseAnnotation or ReservedAnnotation
	Attributes []Attribute // Optional
}

// String returns MF2 formatted string.
func (e Expression) String() string {
	var s []string

	if e.Operand != nil {
		s = append(s, e.Operand.String())
	}

	if e.Annotation != nil {
		s = append(s, e.Annotation.String())
	}

	if len(e.Attributes) > 0 {
		s = append(s, sliceToString(e.Attributes, " "))
	}

	if len(s) == 0 {
		return "{}"
	}

	return "{ " + strings.Join(s, " ") + " }"
}

func (Expression) node()        {}
func (Expression) patternPart() {}

// ---------------------------------Literal------------------------------------

type QuotedLiteral string

// String returns MF2 formatted string.
func (l QuotedLiteral) String() string {
	// quoted-escape = backslash ( backslash / "|" )
	r := strings.NewReplacer(
		`\`, `\\`,
		`|`, `\|`,
	)

	return "|" + r.Replace(string(l)) + "|"
}

func (QuotedLiteral) node()         {}
func (QuotedLiteral) literal()      {}
func (QuotedLiteral) value()        {}
func (QuotedLiteral) variantKey()   {}
func (QuotedLiteral) reservedBody() {}

type NameLiteral string

// String returns MF2 formatted string.
func (l NameLiteral) String() string {
	return string(l)
}

func (NameLiteral) node()       {}
func (NameLiteral) literal()    {}
func (NameLiteral) value()      {}
func (NameLiteral) variantKey() {}

type NumberLiteral float64

// String returns MF2 formatted string.
func (l NumberLiteral) String() string { return strconv.FormatFloat(float64(l), 'f', -1, 64) }

func (NumberLiteral) node()       {}
func (NumberLiteral) literal()    {}
func (NumberLiteral) value()      {}
func (NumberLiteral) variantKey() {}

// --------------------------------Annotation----------------------------------

type Function struct {
	Identifier Identifier
	Options    []Option // Optional
}

// String returns MF2 formatted string.
func (f Function) String() string {
	if len(f.Options) == 0 {
		return ":" + f.Identifier.String()
	}

	return ":" + f.Identifier.String() + " " + sliceToString(f.Options, " ")
}

func (Function) node()       {}
func (Function) annotation() {}

type PrivateUseAnnotation struct {
	ReservedBody []ReservedBody // QuotedLiteral or ReservedText
	Start        rune
}

// String returns MF2 formatted string.
func (p PrivateUseAnnotation) String() string {
	body := sliceToString(p.ReservedBody, " ")
	if len(body) > 0 {
		return string(p.Start) + " " + body
	}

	return string(p.Start)
}

func (PrivateUseAnnotation) node()       {}
func (PrivateUseAnnotation) annotation() {}

type ReservedAnnotation PrivateUseAnnotation

// String returns MF2 formatted string.
func (p ReservedAnnotation) String() string {
	return PrivateUseAnnotation(p).String()
}

func (ReservedAnnotation) node()       {}
func (ReservedAnnotation) annotation() {}

// --------------------------------Declaration---------------------------------

type InputDeclaration Expression // Only VariableExpression, i.e. operand is type Variable.

// String returns MF2 formatted string.
func (d InputDeclaration) String() string {
	return input + " " + Expression(d).String()
}

func (InputDeclaration) node()        {}
func (InputDeclaration) declaration() {}

type LocalDeclaration struct {
	Variable   Variable
	Expression Expression
}

// String returns MF2 formatted string.
func (d LocalDeclaration) String() string {
	return local + " " + d.Variable.String() + " = " + d.Expression.String()
}

func (LocalDeclaration) node()        {}
func (LocalDeclaration) declaration() {}

type ReservedStatement struct {
	Keyword      string
	ReservedBody []ReservedBody // QuotedLiteral or ReservedText
	Expressions  []Expression   // At least one
}

// String returns MF2 formatted string.
func (s ReservedStatement) String() string {
	if len(s.ReservedBody) > 0 {
		return "." + s.Keyword + " " + sliceToString(s.ReservedBody, " ") + " " + sliceToString(s.Expressions, " ")
	}

	return "." + s.Keyword + " " + sliceToString(s.Expressions, " ")
}

func (ReservedStatement) node()        {}
func (ReservedStatement) declaration() {}

// --------------------------------VariantKey----------------------------------

// CatchAllKey is a special key, that matches any value.
type CatchAllKey struct{}

// String returns MF2 formatted string.
func (k CatchAllKey) String() string {
	return catchAllSymbol
}

func (CatchAllKey) node()       {}
func (CatchAllKey) variantKey() {}

// ---------------------------------ComplexBody--------------------------------------

type QuotedPattern []PatternPart

// String returns MF2 formatted string.
func (p QuotedPattern) String() string {
	return "{{" + sliceToString(p, "") + "}}"
}

func (QuotedPattern) node()        {}
func (QuotedPattern) complexBody() {}

type Matcher struct {
	Selectors []Expression // At least one
	Variants  []Variant    // At least one
}

// String returns MF2 formatted string.
func (m Matcher) String() string {
	selectors := sliceToString(m.Selectors, " ")
	variants := sliceToString(m.Variants, "\n")

	return match + " " + selectors + "\n" + variants
}

func (Matcher) node()        {}
func (Matcher) complexBody() {}

// ---------------------------------Node---------------------------------

type Variable string

// String returns MF2 formatted string.
func (v Variable) String() string {
	return string(variablePrefix) + string(v)
}

func (Variable) node()  {}
func (Variable) value() {}

type ReservedText string

// String returns MF2 formatted string.
func (t ReservedText) String() string {
	return strings.NewReplacer(
		`\`, `\\`,
		`{`, `\{`,
		`}`, `\}`,
		`|`, `\|`,
	).Replace(string(t))
}

func (ReservedText) node()         {}
func (ReservedText) reservedBody() {}

type Identifier struct {
	Node

	Namespace string // Optional
	Name      string
}

// String returns MF2 formatted string.
func (i Identifier) String() string {
	if i.Namespace == "" {
		return i.Name
	}

	return i.Namespace + ":" + i.Name
}

type Variant struct {
	Node

	Keys          []VariantKey // At least one: Literal or CatchAllKey
	QuotedPattern QuotedPattern
}

// String returns MF2 formatted string.
func (v Variant) String() string {
	return sliceToString(v.Keys, " ") + " " + v.QuotedPattern.String()
}

type Option struct {
	Node

	Value      Value // Literal or Variable
	Identifier Identifier
}

// String returns MF2 formatted string.
func (o Option) String() string {
	return o.Identifier.String() + " = " + o.Value.String()
}

type MarkupType int

const (
	Unspecified MarkupType = iota
	Open
	Close
	SelfClose
)

type Markup struct {
	PatternPart

	Identifier Identifier
	Options    []Option    // Optional. Options for Identifier, only allowed when markup-open.
	Attributes []Attribute // Optional
	Typ        MarkupType
}

// String returns MF2 formatted string.
func (m Markup) String() string {
	switch m.Typ {
	default:
		return ""
	case Open:
		s := "{ #" + m.Identifier.String()

		if len(m.Options) > 0 {
			s += " " + sliceToString(m.Options, " ")
		}

		if len(m.Attributes) > 0 {
			s += " " + sliceToString(m.Attributes, " ")
		}

		return s + " }"
	case Close:
		s := "{ /" + m.Identifier.String()

		if len(m.Attributes) > 0 {
			s += " " + sliceToString(m.Attributes, " ")
		}

		return s + " }"
	case SelfClose:
		s := "{ #" + m.Identifier.String()

		if len(m.Options) > 0 {
			s += " " + sliceToString(m.Options, " ")
		}

		if len(m.Attributes) > 0 {
			s += " " + sliceToString(m.Attributes, " ")
		}

		return s + " /}"
	}
}

type Attribute struct {
	Node

	Value      Value // Optional: Literal or Variable
	Identifier Identifier
}

// String returns MF2 formatted string.
func (a Attribute) String() string {
	if a.Value == nil {
		return "@" + a.Identifier.String()
	}

	return "@" + a.Identifier.String() + " = " + a.Value.String()
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
