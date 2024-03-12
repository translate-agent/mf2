package template

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"golang.org/x/exp/slices"

	ast "go.expect.digital/mf2/parse"
	"go.expect.digital/mf2/template/registry"
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
	ErrUnsupportedStatement  = errors.New("unsupported statement")
	ErrDuplicateDeclaration  = errors.New("duplicate declaration")
	ErrSelection             = errors.New("selection error")
)

// Func is a function, that will be called when a function is encountered in the template.
type Func func(operand any, options map[string]any) (string, error)

// Template represents a MessageFormat2 template.
type Template struct {
	// TODO: locale field. Can change the output of some functions.
	// e.g. number formatting, given example { $num :number }:
	//  - "en-US" -> 1,234.56
	//  - "lv-LV" -> 1234,56
	// e.g. date formatting, given example { $date :datetime }:
	//  - "en-US" -> 1/2/2023
	//  - "lv-LV" -> 2.1.2023
	ast          *ast.AST
	funcRegistry registry.Registry
}

type Selector struct { //nolint:govet
	Key   string
	Value any
}

// AddFunc adds a function to the template's function map.
func (t *Template) AddFunc(f registry.Func) {
	t.funcRegistry[f.Name] = f
}

// Parse parses the MessageFormat2 string and returns the template.
func (t *Template) Parse(input string) (*Template, error) {
	ast, err := ast.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSyntax, err.Error())
	}

	t.ast = &ast

	return t, nil
}

// Execute writes the result of the template to the given writer.
func (t *Template) Execute(wr io.Writer, input map[string]any) error {
	if t.ast == nil {
		return errors.New("AST is nil")
	}

	executer := &executer{template: t, wr: wr, input: input}

	return executer.execute()
}

// Sprint wraps Execute and returns the result as a string.
func (t *Template) Sprint(input map[string]any) (string, error) {
	sb := new(strings.Builder)

	if err := t.Execute(sb, input); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	return sb.String(), nil
}

// New returns a new Template.
func New() *Template {
	return &Template{funcRegistry: registry.New()}
}

type executer struct {
	template *Template
	wr       io.Writer
	input    map[string]any
}

func (e *executer) write(s string) error {
	if _, err := e.wr.Write([]byte(s)); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (e *executer) execute() error {
	switch message := e.template.ast.Message.(type) {
	default:
		return fmt.Errorf("unknown message type: '%T'", message)
	case nil:
		return nil
	case ast.SimpleMessage:
		for _, pattern := range message {
			if err := e.resolvePattern(pattern); err != nil {
				return fmt.Errorf("resolve pattern: %w", err)
			}
		}
	case ast.ComplexMessage:
		return e.resolveComplexMessage(message)
	}

	return nil
}

func (e *executer) resolveComplexMessage(message ast.ComplexMessage) error {
	if err := e.resolveDeclarations(message.Declarations); err != nil {
		return fmt.Errorf("resolve declarations: %w", err)
	}

	if err := e.resolveComplexBody(message.ComplexBody); err != nil {
		return fmt.Errorf("resolve complex body: %w", err)
	}

	return nil
}

func (e *executer) resolveDeclarations(declarations []ast.Declaration) error {
	m := make(map[ast.Value]struct{}, len(declarations))

	for _, decl := range declarations {
		switch d := decl.(type) {
		case ast.ReservedStatement:
			return fmt.Errorf("%w: '%s'", ErrUnsupportedStatement, "reserved statement")
		case ast.LocalDeclaration:
			if _, ok := m[d.Variable]; ok {
				return fmt.Errorf("%w '%s'", ErrDuplicateDeclaration, d)
			}

			m[d.Variable] = struct{}{}

			resolved, err := e.resolveExpression(d.Expression)
			if err != nil {
				return fmt.Errorf("resolve expression: %w", err)
			}

			e.input[string(d.Variable)] = resolved

		case ast.InputDeclaration:
			if _, ok := m[d.Operand]; ok {
				return fmt.Errorf("%w '%s'", ErrDuplicateDeclaration, d)
			}

			m[d.Operand] = struct{}{}

			resolved, err := e.resolveExpression(ast.Expression(d))
			if err != nil {
				return fmt.Errorf("resolve expression: %w", err)
			}

			e.input[string(d.Operand.(ast.Variable))] = resolved //nolint: forcetypeassert // Will always be a variable.
		}
	}

	return nil
}

func (e *executer) resolveComplexBody(body ast.ComplexBody) error {
	switch b := body.(type) {
	case ast.Matcher:
		result, err := e.resolveMatcher(b)
		if err != nil {
			return fmt.Errorf("resolve matcher: %w", err)
		}

		if err := e.write(result); err != nil {
			return fmt.Errorf("write matcher result: %w", err)
		}
	case ast.QuotedPattern:
		for _, p := range b {
			if err := e.resolvePattern(p); err != nil {
				return fmt.Errorf("resolve pattern: %w", err)
			}
		}
	}

	return nil
}

func (e *executer) resolvePattern(pattern ast.Pattern) error {
	switch patternType := pattern.(type) {
	case ast.TextPattern:
		if err := e.write(string(patternType)); err != nil {
			return fmt.Errorf("write text pattern: %w", err)
		}
	case ast.Expression:
		resolved, err := e.resolveExpression(patternType)
		if err != nil {
			return fmt.Errorf("resolve expression: %w", err)
		}

		if err := e.write(resolved); err != nil {
			return fmt.Errorf("resolve expression: %w", err)
		}
	//nolint:lll
	// When formatting to a string, markup placeholders format to an empty string by default.
	// https://github.com/unicode-org/message-format-wg/blob/main/exploration/open-close-placeholders.md#formatting-to-a-string
	case ast.Markup:
	}

	return nil
}

func (e *executer) resolveExpression(expr ast.Expression) (string, error) {
	value, err := e.resolveValue(expr.Operand)
	if err != nil {
		return "", fmt.Errorf("resolve value: %w", err)
	}

	if expr.Annotation == nil {
		// NOTE: Parser won't allow value to be nil if annotation is nil.
		return fmt.Sprint(value), nil // TODO: If value does not implement fmt.Stringer, what then ?
	}

	resolved, err := e.resolveAnnotation(value, expr.Annotation)
	if err != nil {
		return "", fmt.Errorf("resolve annotation: %w", err)
	}

	return resolved, nil
}

// resolveValue resolves the value of an expression's operand.
//
//   - If the operand is a literal, it returns the literal's value.
//   - If the operand is a variable, it returns the value of the variable from the input map.
func (e *executer) resolveValue(v ast.Value) (any, error) {
	switch v := v.(type) {
	default:
		return nil, fmt.Errorf("unknown value type: '%T'", v)
	case nil:
		return v, nil // nil is also a valid value.
	case ast.QuotedLiteral:
		return string(v), nil
	case ast.NameLiteral:
		return string(v), nil
	case ast.NumberLiteral:
		return float64(v), nil
	case ast.Variable:
		val, ok := e.input[string(v)]
		if !ok {
			return nil, fmt.Errorf("%w '%s'", ErrUnresolvedVariable, v)
		}

		return val, nil
	}
}

func (e *executer) resolveAnnotation(operand any, annotation ast.Annotation) (string, error) {
	annoFn, ok := annotation.(ast.Function)
	if !ok {
		return "", fmt.Errorf("%w with %T annotation: '%s'", ErrUnsupportedExpression, annotation, annotation)
	}

	registryF, ok := e.template.funcRegistry[annoFn.Identifier.Name]
	if !ok {
		return "", fmt.Errorf("%w '%s'", ErrUnknownFunction, annoFn.Identifier.Name)
	}

	opts, err := e.resolveOptions(annoFn.Options)
	if err != nil {
		return "", fmt.Errorf("resolve options: %w", err)
	}

	result, err := registryF.Format(operand, opts)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrFormatting, err.Error())
	}

	return fmt.Sprint(result), nil
}

func (e *executer) resolveOptions(options []ast.Option) (map[string]any, error) {
	m := make(map[string]any, len(options))

	for _, opt := range options {
		name := opt.Identifier.Name
		if _, ok := m[name]; ok {
			return nil, fmt.Errorf("%w '%s'", ErrDuplicateOptionName, name)
		}

		value, err := e.resolveValue(opt.Value)
		if err != nil {
			return nil, fmt.Errorf("resolve value: %w", err)
		}

		m[name] = value
	}

	return m, nil
}

// https://github.com/unicode-org/message-format-wg/blob/main/spec/formatting.md#pattern-selection
func (e *executer) resolveMatcher(m ast.Matcher) (string, error) {
	output, err := e.patternSelection(m)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

func (e *executer) patternSelection(m ast.Matcher) (strings.Builder, error) {
	res, err := e.resolveSelector()
	if err != nil {
		return strings.Builder{}, err
	}

	pref := e.resolvePreferences(m, res)

	filteredVariants := e.filterVariants(m, pref)

	sortable := e.sortVariants(filteredVariants, pref)

	output, err := e.selectBestVariant(sortable)
	if err != nil {
		return strings.Builder{}, err
	}

	return output, nil
}

func (e *executer) resolveSelector() ([]Selector, error) {
	// Step 1: Resolve Selector Value (Modified)
	var res []Selector

	for k, v := range e.input {
		rv := Selector{k, v}
		if e.input[k] != "" {
			res = append(res, rv)
		} else {
			return nil, fmt.Errorf("%w '%s'", ErrSelection, res)
		}
	}

	return res, nil
}

func (e *executer) resolvePreferences(m ast.Matcher, res []Selector) [][]string {
	// Step 2: Resolve Preferences
	var pref [][]string //nolint:prealloc

	for i := range res {
		var keys []string

		for _, variant := range m.Variants {
			for _, vKey := range variant.Keys {
				key := vKey
				if key.String() != "*" {
					keys = append(keys, key.String())
				}
			}
		}

		rv := res[i]

		matches := matchSelectorKeys(rv, keys)
		pref = append(pref, matches)
	}

	return pref
}

func (e *executer) filterVariants(m ast.Matcher, pref [][]string) []ast.Variant {
	// Step 3: Filter Variants
	var filteredVariants []ast.Variant

	for _, variant := range m.Variants {
		matchesAllSelectors := true

		for i, keyOrder := range pref {
			key := variant.Keys[i]
			if key.String() == "*" {
				continue
			}

			ks := key.String()
			if !slices.Contains(keyOrder, ks) {
				matchesAllSelectors = false
				break
			}
		}

		if matchesAllSelectors {
			filteredVariants = append(filteredVariants, variant)
		}
	}

	return filteredVariants
}

func (e *executer) sortVariants(filteredVariants []ast.Variant, pref [][]string) []SortableVariant {
	// Step 4: Sort Variants
	sortable := make([]SortableVariant, 0, len(filteredVariants))

	for _, variant := range filteredVariants {
		sortable = append(sortable, SortableVariant{Score: -1, Variant: variant})
	}

	for i := len(pref) - 1; i >= 0; i-- {
		matches := pref[i]

		for tupleIndex, tuple := range sortable {
			key := tuple.Variant.Keys[i]
			currentScore := len(matches)

			if key.String() != "*" {
				ks := key.String()
				if position := findPosition(ks, matches); position != -1 {
					currentScore = position
				}
			}

			sortable[tupleIndex].Score = currentScore
		}

		SortVariants(sortable)
	}

	return sortable
}

func (e *executer) selectBestVariant(sortable []SortableVariant) (strings.Builder, error) { //nolint:unparam
	// Select the best variant
	bestVariant := sortable[0].Variant

	var output strings.Builder

	for _, patternElement := range bestVariant.QuotedPattern {
		if err := e.resolvePattern(patternElement); err != nil {
			return strings.Builder{}, fmt.Errorf("resolve pattern element: %w", err)
		}
	}

	return output, nil
}

// The SortVariants function.
func SortVariants(sortable []SortableVariant) {
	sort.Sort(SortableVariantSlice(sortable))
}

func matchSelectorKeys(rv Selector, keys []string) []string {
	var matches []string

	for _, key := range keys {
		if value, ok := rv.Value.(string); ok {
			if key == value {
				matches = append(matches, key)
			}
		}
	}

	return matches
}

type SortableVariant struct {
	Variant ast.Variant
	Score   int
}

type SortableVariantSlice []SortableVariant

func (s SortableVariantSlice) Len() int {
	return len(s)
}

func (s SortableVariantSlice) Less(i, j int) bool {
	return s[i].Score < s[j].Score
}

func (s SortableVariantSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func findPosition(ks string, matches []string) int {
	for index, match := range matches {
		if ks == match {
			return index
		}
	}

	return -1 // Not found
}
