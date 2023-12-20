package mf2

import (
	"errors"
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

/*
Validate returns an error if the AST is invalid according to the MessageFormat 2.0 specification.
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
				PlaceholderPattern{
					Expression: VariableExpression{Variable: Variable("")},
				},
				TextPattern(" World!"),
			},
		},
	},

	err := ast.Validate() // err: ast.message.patterns.placeholderPattern.expression.variable: name is empty '{ $ }'
*/
func (a AST) Validate() error {
	if a.Message == nil {
		return errors.New("ast: message is required")
	}

	if err := a.Message.Validate(); err != nil {
		return fmt.Errorf("ast.%w", err)
	}

	return nil
}

// --------------------------------Interfaces----------------------------------
//
// Here we define the Nodes that can have multiple types.
// For example Message could be either a SimpleMessage or a ComplexMessage.
// Pattern could be either a TextPattern or a PlaceholderPattern.
// etc.

// Node is the interface implemented by all AST nodes.
type Node interface {
	node()
	Validate() error // Validate returns an error if the Node is invalid according to the MessageFormat 2.0 specification.

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

func (sm SimpleMessage) Validate() error {
	if err := validateSlice(sm.Patterns); err != nil {
		return fmt.Errorf("simpleMessage.%w", err)
	}

	return nil
}

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

func (cm ComplexMessage) Validate() error {
	if cm.ComplexBody == nil {
		return errors.New("complexMessage: complexBody is required")
	}

	if err := cm.ComplexBody.Validate(); err != nil {
		return fmt.Errorf("complexMessage.%w", err)
	}

	if err := validateSlice(cm.Declarations); err != nil {
		return fmt.Errorf("complexMessage.%w", err)
	}

	return nil
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
func (tp TextPattern) Validate() error { return nil }

type PlaceholderPattern struct {
	Pattern

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
}

func (pp PlaceholderPattern) String() string { return fmt.Sprint(pp.Expression) }

func (pp PlaceholderPattern) Validate() error {
	if pp.Expression == nil {
		return errors.New("placeholderPattern: expression is required")
	}

	if err := pp.Expression.Validate(); err != nil {
		return fmt.Errorf("placeholderPattern.%w", err)
	}

	return nil
}

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

func (le LiteralExpression) Validate() error {
	if le.Literal == nil {
		return errors.New("literalExpression: literal is required")
	}

	if err := le.Literal.Validate(); err != nil {
		return fmt.Errorf("literalExpression:.%w", err)
	}

	if le.Annotation == nil {
		return nil
	}

	if err := le.Annotation.Validate(); err != nil {
		return fmt.Errorf("literalExpression.%w", err)
	}

	return nil
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

func (ve VariableExpression) Validate() error {
	if err := ve.Variable.Validate(); err != nil {
		return fmt.Errorf("variableExpression.%w", err)
	}

	if ve.Annotation == nil {
		return nil
	}

	if err := ve.Annotation.Validate(); err != nil {
		return fmt.Errorf("variableExpression.%w", err)
	}

	return nil
}

type AnnotationExpression struct {
	Expression

	Annotation Annotation // FunctionAnnotation, PrivateUseAnnotation, or ReservedAnnotation
}

func (ae AnnotationExpression) String() string { return fmt.Sprintf("{ %s }", ae.Annotation) }

func (ae AnnotationExpression) Validate() error {
	if ae.Annotation == nil {
		return errors.New("annotationExpression: annotation is required")
	}

	if err := ae.Annotation.Validate(); err != nil {
		return fmt.Errorf("annotationExpression.%w", err)
	}

	return nil
}

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

func (ql QuotedLiteral) Validate() error {
	if isZeroValue(ql) {
		return errors.New("quotedLiteral: literal is empty")
	}

	return nil
}

type UnquotedLiteral struct {
	Literal

	Value Unquoted // NameLiteral or NumberLiteral
}

func (ul UnquotedLiteral) String() string { return fmt.Sprint(ul.Value) }
func (ul UnquotedLiteral) Validate() error {
	if ul.Value == nil {
		return errors.New("unquotedLiteral: literal is empty")
	}

	if err := ul.Value.Validate(); err != nil {
		return fmt.Errorf("unquotedLiteral.%w", err)
	}

	return nil
}

type NameLiteral string

func (NameLiteral) node()             {}
func (NameLiteral) literal()          {}
func (NameLiteral) unquoted()         {}
func (nl NameLiteral) String() string { return string(nl) }
func (nl NameLiteral) Validate() error {
	if isZeroValue(nl) {
		return errors.New("nameLiteral: literal is empty")
	}

	return nil
}

type NumberLiteral float64

func (NumberLiteral) node()              {}
func (NumberLiteral) literal()           {}
func (NumberLiteral) unquoted()          {}
func (nl NumberLiteral) String() string  { return fmt.Sprint(float64(nl)) }
func (nl NumberLiteral) Validate() error { return nil } // Zero value is valid

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

func (fa FunctionAnnotation) Validate() error {
	if err := fa.Function.Validate(); err != nil {
		return fmt.Errorf("functionAnnotation.%w", err)
	}

	if len(fa.Options) == 0 {
		return nil
	}

	if err := validateSlice(fa.Options); err != nil {
		return fmt.Errorf("functionAnnotation.%w", err)
	}

	return nil
}

type PrivateUseAnnotation struct {
	Annotation

	// TODO: Implementation
}

func (PrivateUseAnnotation) String() string  { return "TODO" }
func (PrivateUseAnnotation) Validate() error { return nil }

type ReservedAnnotation struct {
	Annotation

	// TODO: Implementation
}

func (ReservedAnnotation) String() string  { return "TODO" }
func (ReservedAnnotation) Validate() error { return nil }

// ---------------------------------Option-------------------------------------

type LiteralOption struct {
	Option

	Literal    Literal // QuotedLiteral or UnquotedLiteral
	Identifier Identifier
}

func (lo LiteralOption) String() string { return fmt.Sprintf("%s = %s", lo.Identifier, lo.Literal) }

func (lo LiteralOption) Validate() error {
	if lo.Literal == nil {
		return errors.New("literalOption: literal is required")
	}

	if err := lo.Literal.Validate(); err != nil {
		return fmt.Errorf("literalOption.%w", err)
	}

	if err := lo.Identifier.Validate(); err != nil {
		return fmt.Errorf("literalOption.%w", err)
	}

	return nil
}

type VariableOption struct {
	Option

	Identifier Identifier
	Variable   Variable
}

func (vo VariableOption) String() string { return fmt.Sprintf("%s = %s", vo.Identifier, vo.Variable) }

func (vo VariableOption) Validate() error {
	if err := vo.Variable.Validate(); err != nil {
		return fmt.Errorf("variableOption.%w", err)
	}

	if err := vo.Identifier.Validate(); err != nil {
		return fmt.Errorf("variableOption.%w", err)
	}

	return nil
}

// --------------------------------Declaration---------------------------------

type InputDeclaration struct {
	Declaration

	Expression VariableExpression
}

func (id InputDeclaration) String() string { return fmt.Sprintf("%s %s", input, id.Expression) }
func (id InputDeclaration) Validate() error {
	if err := id.Expression.Validate(); err != nil {
		return fmt.Errorf("inputDeclaration.%w", err)
	}

	return nil
}

type LocalDeclaration struct {
	Declaration

	Expression Expression // LiteralExpression, VariableExpression, or AnnotationExpression
	Variable   Variable
}

func (ld LocalDeclaration) String() string {
	return fmt.Sprintf("%s %s = %s", local, ld.Variable, ld.Expression)
}

func (ld LocalDeclaration) Validate() error {
	if ld.Expression == nil {
		return errors.New("localDeclaration: expression is required")
	}

	if err := ld.Expression.Validate(); err != nil {
		return fmt.Errorf("localDeclaration.%w", err)
	}

	if err := ld.Expression.Validate(); err != nil {
		return fmt.Errorf("localDeclaration.%w", err)
	}

	return nil
}

type ReservedStatement struct {
	Declaration

	// TODO: Implementation
}

func (ReservedStatement) String() string  { return "TODO" }
func (ReservedStatement) Validate() error { return nil }

// --------------------------------VariantKey----------------------------------

type LiteralKey struct {
	VariantKey

	Literal Literal // QuotedLiteral or UnquotedLiteral
}

func (lk LiteralKey) String() string { return fmt.Sprint(lk.Literal) }
func (lk LiteralKey) Validate() error {
	if lk.Literal == nil {
		return errors.New("literalKey: literal is required")
	}

	if err := lk.Literal.Validate(); err != nil {
		return fmt.Errorf("literalKey.%w", err)
	}

	return nil
}

// CatchAllKey is a special key, that matches any value.
type CatchAllKey struct {
	VariantKey
}

func (ck CatchAllKey) String() string  { return catchAllSymbol }
func (ck CatchAllKey) Validate() error { return nil }

// ---------------------------------ComplexBody--------------------------------------

type QuotedPattern struct {
	ComplexBody

	Patterns []Pattern // TextPattern or PlaceholderPattern
}

func (qp QuotedPattern) String() string {
	return fmt.Sprintf("{{%s}}", sliceToString(qp.Patterns, ""))
}

func (qp QuotedPattern) Validate() error {
	if err := validateSlice(qp.Patterns); err != nil {
		return fmt.Errorf("quotedPattern.%w", err)
	}

	return nil
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

func (m Matcher) Validate() error {
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

// ---------------------------------Node---------------------------------

type Variable string

func (Variable) node()            {}
func (v Variable) String() string { return fmt.Sprintf("%c%s", variablePrefix, string(v)) }
func (v Variable) Validate() error {
	if isZeroValue(v) {
		return errors.New("variable: name is empty")
	}

	return nil
}

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

func (i Identifier) Validate() error {
	if isZeroValue(i.Name) {
		return errors.New("identifier: name is empty")
	}

	return nil
}

type Function struct {
	Node

	Identifier Identifier
	Prefix     rune // One of: ':', '+', '-'
}

func (f Function) String() string { return fmt.Sprintf("%c%s", f.Prefix, f.Identifier) }
func (f Function) Validate() error {
	if err := f.Identifier.Validate(); err != nil {
		return fmt.Errorf("function.%w", err)
	}

	switch f.Prefix {
	case ':', '+', '-':
	default:
		return fmt.Errorf("function: invalid prefix: %q", f.Prefix)
	}

	return nil
}

type Variant struct {
	Node

	Keys          []VariantKey // At least one: LiteralKey or WildcardKey
	QuotedPattern QuotedPattern
}

func (v Variant) String() string {
	return fmt.Sprintf("%s %s", sliceToString(v.Keys, " "), v.QuotedPattern)
}

func (v Variant) Validate() error {
	if len(v.Keys) == 0 {
		return errors.New("variant: at least one key is required")
	}

	if err := validateSlice(v.Keys); err != nil {
		return fmt.Errorf("variant.%w", err)
	}

	if err := v.QuotedPattern.Validate(); err != nil {
		return fmt.Errorf("variant.%w", err)
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
	nodeStrings := make([]string, len(s))
	for i, node := range s {
		nodeStrings[i] = fmt.Sprint(node)
	}

	return strings.Join(nodeStrings, sep)
}

// isZeroValue returns true if v is the zero value of its type.
func isZeroValue[T comparable](v T) bool {
	var zero T

	return v == zero
}

// validateSlice validates a slice of Nodes.
func validateSlice[T Node](s []T) error {
	for _, v := range s {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("%w '%s'", err, fmt.Sprint(v))
		}
	}

	return nil
}
