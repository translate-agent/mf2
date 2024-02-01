package parse

import (
	"errors"
	"fmt"
	"math"
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
				Expression{Operand: Variable("variable")}
				TextPattern(" World!"),
			},
		},
	}

	fmt.Print(ast) // Hello, { $variable } World!
*/
func (a AST) String() string { return fmt.Sprint(a.Message) }

/*
validate returns an error if the AST is invalid according to the MessageFormat 2.0 specification.
For example, when matcher has no selectors or variants.
Or variable is zero value, i.e $.

If one of the nodes is invalid, the error will contain path to the node which failed validation, and
the string representation of the node.

Example:

	// Hello, { $ } World! // MF2 formatted message
	ast := AST{
		Message: SimpleMessage{
			Patterns: []Pattern{
				TextPattern("Hello, "),
				Expression{Operand: Variable("")},
				TextPattern(" World!"),
			},
		},
	},

	err := ast.validate() // err: ast.simpleMessage.expression.variable: name is empty '{ $}'
*/
func (a AST) validate() error {
	if a.Message == nil {
		return errors.New("ast: message is required")
	}

	if err := a.Message.validate(); err != nil {
		return fmt.Errorf("ast.%w", err)
	}

	return nil
}

// --------------------------------Interfaces----------------------------------
//
// Here we define the Nodes that can have multiple types.
// For example Message could be either a SimpleMessage or a ComplexMessage.
// Pattern could be either a TextPattern, Expression or a Markup.
// etc.

// Node is the interface implemented by all AST nodes.
type Node interface {
	node()
	validate() error // validate returns an error if the Node is invalid according to the MessageFormat 2.0 specification.

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

type SimpleMessage []Pattern

type ComplexMessage struct {
	ComplexBody  ComplexBody   // Matcher or QuotedPattern
	Declarations []Declaration // Optional: InputDeclaration, LocalDeclaration or ReservedStatement
}

func (sm SimpleMessage) String() string { return sliceToString(sm, "") }
func (cm ComplexMessage) String() string {
	if len(cm.Declarations) == 0 {
		return fmt.Sprint(cm.ComplexBody)
	}

	return fmt.Sprintf("%s\n%s", sliceToString(cm.Declarations, "\n"), cm.ComplexBody)
}

func (sm SimpleMessage) validate() error {
	if err := validateSlice(sm); err != nil {
		return fmt.Errorf("simpleMessage.%w", err)
	}

	return nil
}

func (cm ComplexMessage) validate() error {
	if cm.ComplexBody == nil {
		return errors.New("complexMessage: complexBody is required")
	}

	if err := cm.ComplexBody.validate(); err != nil {
		return fmt.Errorf("complexMessage.%w", err)
	}

	if err := validateSlice(cm.Declarations); err != nil {
		return fmt.Errorf("complexMessage.%w", err)
	}

	return nil
}

func (sm SimpleMessage) message()  {}
func (cm ComplexMessage) message() {}

func (sm SimpleMessage) node()  {}
func (cm ComplexMessage) node() {}

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
func (tp TextPattern) validate() error { return nil }

// --------------------------------Expression----------------------------------

type Expression struct {
	Operand    Value       // Literal or Variable
	Annotation Annotation  // Function, PrivateUseAnnotation or ReservedAnnotation
	Attributes []Attribute // Optional
}

func (e Expression) String() string {
	var s string

	if e.Operand != nil {
		s = fmt.Sprintf(" %s", e.Operand)
	}

	if e.Annotation != nil {
		s += fmt.Sprintf(" %s", e.Annotation)
	}

	if len(e.Attributes) > 0 {
		s += fmt.Sprintf(" %s", sliceToString(e.Attributes, " "))
	}

	return fmt.Sprintf("{%s}", s)
}

func (e Expression) validate() error {
	if e.Operand == nil && e.Annotation == nil {
		return errors.New("expression: at least one of operand or annotation is required")
	}

	if e.Operand != nil {
		if err := e.Operand.validate(); err != nil {
			return fmt.Errorf("expression.%w", err)
		}
	}

	if e.Annotation != nil {
		if err := e.Annotation.validate(); err != nil {
			return fmt.Errorf("expression.%w", err)
		}
	}

	if err := validateSlice(e.Attributes); err != nil {
		return fmt.Errorf("expression.%w", err)
	}

	return nil
}

func (Expression) node()    {}
func (Expression) pattern() {}

// ---------------------------------Literal------------------------------------

type QuotedLiteral string

type NameLiteral string

type NumberLiteral float64

func (ql QuotedLiteral) String() string {
	// quoted-escape = backslash ( backslash / "|" )
	r := strings.NewReplacer(
		`\`, `\\`,
		`|`, `\|`,
	)

	return fmt.Sprintf("|%s|", r.Replace(string(ql)))
}
func (nl NameLiteral) String() string   { return string(nl) }
func (nl NumberLiteral) String() string { return fmt.Sprint(float64(nl)) }

func (ql QuotedLiteral) validate() error {
	if isZeroValue(ql) {
		return errors.New("quotedLiteral: literal is empty")
	}

	return nil
}

func (nl NameLiteral) validate() error {
	if isZeroValue(nl) {
		return errors.New("nameLiteral: literal is empty")
	}

	return nil
}

func (nl NumberLiteral) validate() error {
	switch {
	case math.IsInf(float64(nl), 0):
		return errors.New("numberLiteral: literal is infinite")
	case math.IsNaN(float64(nl)):
		return errors.New("numberLiteral: literal is NaN")
	default:
		return nil
	}
}

func (QuotedLiteral) node() {}
func (NameLiteral) node()   {}
func (NumberLiteral) node() {}

func (QuotedLiteral) literal() {}
func (NameLiteral) literal()   {}
func (NumberLiteral) literal() {}

func (QuotedLiteral) value() {}
func (NameLiteral) value()   {}
func (NumberLiteral) value() {}

func (QuotedLiteral) variantKey() {}
func (NameLiteral) variantKey()   {}
func (NumberLiteral) variantKey() {}

func (QuotedLiteral) reservedBody() {}

// --------------------------------Annotation----------------------------------

type Function struct {
	Identifier Identifier
	Options    []Option // Optional
}

type PrivateUseAnnotation struct {
	ReservedBody []ReservedBody // QuotedLiteral or ReservedText
	Start        rune
}

type ReservedAnnotation PrivateUseAnnotation

func (f Function) String() string {
	if len(f.Options) == 0 {
		return fmt.Sprintf(":%s", f.Identifier)
	}

	return fmt.Sprintf(":%s %s", f.Identifier, sliceToString(f.Options, " "))
}

func (p PrivateUseAnnotation) String() string {
	return fmt.Sprintf("%c%s", p.Start, sliceToString(p.ReservedBody, ""))
}

func (p ReservedAnnotation) String() string { return PrivateUseAnnotation(p).String() }

func (f Function) validate() error {
	if err := f.Identifier.validate(); err != nil {
		return fmt.Errorf("function.%w", err)
	}

	if err := validateSlice(f.Options); err != nil {
		return fmt.Errorf("function.%w", err)
	}

	return nil
}

func (p PrivateUseAnnotation) validate() error {
	if !isPrivateStart(p.Start) {
		return fmt.Errorf("privateUseAnnotation: start must be a private start char, got '%c'", p.Start)
	}

	if p.ReservedBody != nil {
		if err := validateSlice(p.ReservedBody); err != nil {
			return fmt.Errorf("privateUseAnnotation.%w", err)
		}
	}

	return nil
}

func (p ReservedAnnotation) validate() error {
	if !isReservedStart(p.Start) {
		return fmt.Errorf("reservedAnnotation: start must be a reserved start char, got '%c'", p.Start)
	}

	if p.ReservedBody != nil {
		if err := validateSlice(p.ReservedBody); err != nil {
			return fmt.Errorf("reservedAnnotation.%w", err)
		}
	}

	return nil
}

func (Function) node()             {}
func (PrivateUseAnnotation) node() {}
func (ReservedAnnotation) node()   {}

func (Function) annotation()             {}
func (PrivateUseAnnotation) annotation() {}
func (ReservedAnnotation) annotation()   {}

// --------------------------------Declaration---------------------------------

type InputDeclaration Expression // Only VariableExpression, i.e. operand is type Variable.

type LocalDeclaration struct {
	Variable   Variable
	Expression Expression
}

type ReservedStatement struct {
	// TODO: Implementation
}

func (id InputDeclaration) String() string { return fmt.Sprintf("%s %s", input, Expression(id)) }
func (ld LocalDeclaration) String() string {
	return fmt.Sprintf("%s %s = %s", local, ld.Variable, ld.Expression)
}
func (ReservedStatement) String() string { return ".RESERVED STATEMENT_NOT_IMPLEMENTED { TODO }" } // TODO: Implement

func (id InputDeclaration) validate() error {
	if id.Operand == nil {
		return errors.New("inputDeclaration: expression operand is required")
	}

	if _, ok := id.Operand.(Variable); !ok {
		return fmt.Errorf("inputDeclaration: expression operand must be a variable, got '%T'", id.Operand)
	}

	if err := Expression(id).validate(); err != nil {
		return fmt.Errorf("inputDeclaration.%w", err)
	}

	return nil
}

func (ld LocalDeclaration) validate() error {
	if err := ld.Expression.validate(); err != nil {
		return fmt.Errorf("localDeclaration.%w", err)
	}

	if err := ld.Expression.validate(); err != nil {
		return fmt.Errorf("localDeclaration.%w", err)
	}

	return nil
}

func (ReservedStatement) validate() error { return nil }

func (InputDeclaration) node()  {}
func (LocalDeclaration) node()  {}
func (ReservedStatement) node() {}

func (InputDeclaration) declaration()  {}
func (LocalDeclaration) declaration()  {}
func (ReservedStatement) declaration() {}

// --------------------------------VariantKey----------------------------------

// CatchAllKey is a special key, that matches any value.
type CatchAllKey struct{}

func (ck CatchAllKey) String() string  { return catchAllSymbol }
func (ck CatchAllKey) validate() error { return nil }
func (CatchAllKey) node()              {}
func (CatchAllKey) variantKey()        {}

// ---------------------------------ComplexBody--------------------------------------

type QuotedPattern []Pattern

type Matcher struct {
	MatchStatements []Expression // At least one
	Variants        []Variant    // At least one
}

func (qp QuotedPattern) String() string { return fmt.Sprintf("{{%s}}", sliceToString(qp, "")) }
func (m Matcher) String() string {
	matchStr := sliceToString(m.MatchStatements, " ")
	variantsStr := sliceToString(m.Variants, "\n")

	return fmt.Sprintf("%s %s\n%s", match, matchStr, variantsStr)
}

func (qp QuotedPattern) validate() error {
	if err := validateSlice(qp); err != nil {
		return fmt.Errorf("quotedPattern.%w", err)
	}

	return nil
}

func (m Matcher) validate() error {
	if len(m.MatchStatements) == 0 {
		return errors.New("matcher: at least one match statement is required")
	}

	if len(m.Variants) == 0 {
		return errors.New("matcher: at least one variant is required")
	}

	if err := validateSlice(m.MatchStatements); err != nil {
		return fmt.Errorf("matcher.%w", err)
	}

	if err := validateSlice(m.Variants); err != nil {
		return fmt.Errorf("matcher.%w", err)
	}

	return nil
}

func (QuotedPattern) node() {}
func (Matcher) node()       {}

func (QuotedPattern) complexBody() {}
func (Matcher) complexBody()       {}

// ---------------------------------Node---------------------------------

type Variable string

func (Variable) node()            {}
func (v Variable) String() string { return fmt.Sprintf("%c%s", variablePrefix, string(v)) }
func (Variable) value()           {}
func (v Variable) validate() error {
	if isZeroValue(v) {
		return errors.New("variable: name is empty")
	}

	return nil
}

type ReservedText string

func (rt ReservedText) String() string {
	return strings.NewReplacer(
		`\`, `\\`,
		`{`, `\{`,
		`}`, `\}`,
		`|`, `\|`,
	).Replace(string(rt))
}

func (rt ReservedText) validate() error {
	if isZeroValue(string(rt)) {
		return errors.New("reservedText: text is empty")
	}

	return nil
}

func (ReservedText) node()         {}
func (ReservedText) reservedBody() {}

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

func (i Identifier) validate() error {
	if isZeroValue(i.Name) {
		return errors.New("identifier: name is empty")
	}

	return nil
}

type Variant struct {
	Node

	Keys          []VariantKey // At least one: Literal or CatchAllKey
	QuotedPattern QuotedPattern
}

func (v Variant) String() string {
	return fmt.Sprintf("%s %s", sliceToString(v.Keys, " "), v.QuotedPattern)
}

func (v Variant) validate() error {
	if len(v.Keys) == 0 {
		return errors.New("variant: at least one key is required")
	}

	if err := validateSlice(v.Keys); err != nil {
		return fmt.Errorf("variant.%w", err)
	}

	if err := v.QuotedPattern.validate(); err != nil {
		return fmt.Errorf("variant.%w", err)
	}

	return nil
}

type Option struct {
	Node

	Value      Value // Literal or Variable
	Identifier Identifier
}

func (o Option) String() string { return fmt.Sprintf("%s = %s", o.Identifier, o.Value) }
func (o Option) validate() error {
	if err := o.Identifier.validate(); err != nil {
		return fmt.Errorf("option.%w", err)
	}

	if o.Value == nil {
		return errors.New("option: value is required")
	}

	if err := o.Value.validate(); err != nil {
		return fmt.Errorf("option.%w", err)
	}

	return nil
}

type MarkupType int

const (
	Unspecified MarkupType = iota
	Open
	Close
	SelfClose
)

type Markup struct {
	Pattern

	Identifier Identifier
	Options    []Option    // Optional. Options for Identifier, only allowed when markup-open.
	Attributes []Attribute // Optional
	Typ        MarkupType
}

func (m Markup) String() string {
	switch m.Typ {
	default:
		return ""
	case Open:
		return fmt.Sprintf("{ #%s %s %s }", m.Identifier, sliceToString(m.Options, " "), sliceToString(m.Attributes, " "))
	case Close:
		return fmt.Sprintf("{ /%s %s }", m.Identifier, sliceToString(m.Attributes, " "))
	case SelfClose:
		return fmt.Sprintf("{ #%s %s %s /}", m.Identifier, sliceToString(m.Options, " "), sliceToString(m.Attributes, " "))
	}
}

func (m Markup) validate() error {
	if err := m.Identifier.validate(); err != nil {
		return fmt.Errorf("markup.%w", err)
	}

	if m.Typ == Close && len(m.Options) != 0 {
		return errors.New("markup: options are not allowed for markup-close")
	}

	if err := validateSlice(m.Options); err != nil {
		return fmt.Errorf("markup.%w", err)
	}

	if err := validateSlice(m.Attributes); err != nil {
		return fmt.Errorf("markup.%w", err)
	}

	return nil
}

type Attribute struct {
	Node

	Value      Value // Optional: Literal or Variable
	Identifier Identifier
}

func (a Attribute) String() string {
	if a.Value == nil {
		return fmt.Sprintf("@%s", a.Identifier)
	}

	return fmt.Sprintf("@%s = %s", a.Identifier, a.Value)
}

func (a Attribute) validate() error {
	if err := a.Identifier.validate(); err != nil {
		return fmt.Errorf("attribute.%w", err)
	}

	if a.Value == nil {
		return nil
	}

	if err := a.Value.validate(); err != nil {
		return fmt.Errorf("attribute.%w", err)
	}

	return nil
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

// isZeroValue returns true if v is the zero value of its type.
func isZeroValue[T comparable](v T) bool {
	var zero T

	return v == zero
}

// validateSlice validates a slice of Nodes.
func validateSlice[T Node](s []T) error {
	for _, v := range s {
		if err := v.validate(); err != nil {
			return fmt.Errorf("%w '%s'", err, fmt.Sprint(v))
		}
	}

	return nil
}
