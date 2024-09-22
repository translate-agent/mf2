package template

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
	"golang.org/x/text/language"

	"go.expect.digital/mf2"
	ast "go.expect.digital/mf2/parse"
)

// Template represents a MessageFormat2 template.
type Template struct {
	ast      *ast.AST
	registry Registry
	locale   language.Tag
}

// ResolvedValue keeps the result of the Expression resolution with optionally
// defined format() and selectKey() functions for Format and Select contexts.
type ResolvedValue struct {
	value     any
	selectKey func(keys []string) string
	format    func() string
	err       error
}

// defaultFormat returns formatted string value for any type.
func defaultFormat(value any) string {
	switch v := value.(type) {
	default:
		return fmt.Sprint(v)
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case []byte:
		return string(v)
	case []rune:
		return string(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.Itoa(int(v))
	case int16:
		return strconv.Itoa(int(v))
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.Itoa(int(v))
	case uint: // byte
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
}

// String returns formatted string value.
func (r *ResolvedValue) String() string {
	if r.format != nil {
		return r.format()
	}

	return defaultFormat(r.value)
}

// ResolvedValueOpt is a function to apply to the [ResolvedValue].
type ResolvedValueOpt func(*ResolvedValue)

// WithFormat applies a formatting function to the [ResolvedValue] returned by [Func].
// The formatting function is called in the formatting context.
func WithFormat(format func() string) ResolvedValueOpt {
	return func(r *ResolvedValue) {
		r.format = format
	}
}

// WithSelectKey applies a selection function to the [ResolvedValue] returned by [Func].
// The selection function is called in the selection context.
//
// Keys exclude catch all key "*". If keys contain "*", it is string literal and is NOT catch all key.
func WithSelectKey(selectKey func(keys []string) string) ResolvedValueOpt {
	return func(r *ResolvedValue) {
		r.selectKey = selectKey
	}
}

// NewResolvedValue creates a new variable of type [*ResolvedValue].
// If value is already [*ResolvedValue], the optional format() and selectKey() are applied to it.
func NewResolvedValue(value any, options ...ResolvedValueOpt) *ResolvedValue {
	r, ok := value.(*ResolvedValue)
	if !ok {
		r = &ResolvedValue{value: value}
	}

	for _, f := range options {
		f(r)
	}

	return r
}

func newFallbackValue(expr ast.Expression) *ResolvedValue {
	wrap := func(v string) *ResolvedValue {
		return NewResolvedValue("{" + v + "}")
	}

	switch v := expr.Operand.(type) {
	default:
		return wrap("\ufffd") // the U+FFFD REPLACEMENT CHARACTER �
	case nil:
		switch f := expr.Annotation.(type) {
		default:
			return wrap(f.String())
		case ast.Function:
			return wrap(":" + f.Identifier.String())
		}
	case ast.QuotedLiteral:
		return wrap(v.String())
	case ast.NameLiteral:
		return wrap(ast.QuotedLiteral(v).String())
	case ast.NumberLiteral:
		return wrap(ast.QuotedLiteral(v.String()).String())
	case ast.Variable:
		return wrap(v.String())
	}
}

// New returns a new Template.
func New(options ...Option) *Template {
	t := &Template{
		registry: NewRegistry(),
		locale:   language.AmericanEnglish,
	}

	for _, o := range options {
		o(t)
	}

	return t
}

// Option is a template option.
type Option func(t *Template)

// WithFunc adds a single function to function registry.
//
// Example:
//
//	WithFunc("bar", f)     // function ":bar"
//	WithFunc("foo:bar", f) // function with namespace ":foo:bar"
func WithFunc(name string, f Func) Option {
	return func(t *Template) {
		t.registry[name] = f
	}
}

// WithFuncs adds functions to function registry.
//
// Example:
//
//	WithFuncs(Registry{
//		"bar": f,     // function ":bar"
//		"foo:bar": f, // function with namespace ":foo:bar"
//	})
func WithFuncs(reg Registry) Option {
	return func(t *Template) {
		for k, f := range reg {
			t.registry[k] = f
		}
	}
}

// WithLocale adds locale information to the template.
func WithLocale(locale language.Tag) Option {
	return func(t *Template) {
		t.locale = locale
	}
}

// Parse parses the MessageFormat2 string and returns the template.
func (t *Template) Parse(input string) (*Template, error) {
	ast, err := ast.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("parse to execute: %w", err)
	}

	t.ast = &ast

	return t, nil
}

// Execute writes the result of the template to the given writer.
func (t *Template) Execute(w io.Writer, input map[string]any) error {
	if t.ast == nil {
		return errors.New("execute template: AST is nil")
	}

	executer := &executer{template: t, w: w, variables: make(map[string]*ResolvedValue, len(input))}

	for k, v := range input {
		var f Func

		switch v.(type) {
		default:
			executer.variables[k] = NewResolvedValue(v, WithFormat(func() string { return defaultFormat(v) }))
			continue
		case string:
			f = stringFunc
		case float64, int:
			f = numberFunc
		}

		r, err := f(NewResolvedValue(v), nil, t.locale)
		if err != nil {
			return fmt.Errorf("execute template: %w", err)
		}

		executer.variables[k] = r
	}

	if err := executer.execute(); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

// Sprint wraps Execute and returns the result as a string.
func (t *Template) Sprint(input map[string]any) (string, error) {
	sb := new(strings.Builder)
	err := t.Execute(sb, input)

	return sb.String(), err
}

type executer struct {
	template  *Template
	w         io.Writer
	variables map[string]*ResolvedValue
}

func (e *executer) execute() error {
	switch message := e.template.ast.Message.(type) {
	default: // this should never happen, AST must be valid.
		return fmt.Errorf("unexpected message type: '%T'", message)
	case nil:
		return nil
	case ast.SimpleMessage:
		return e.resolvePattern(message)
	case ast.ComplexMessage:
		return e.resolveComplexMessage(message)
	}
}

func (e *executer) resolveComplexMessage(message ast.ComplexMessage) error {
	var resolutionErr error

	err := e.resolveDeclarations(message.Declarations)

	switch {
	case errors.Is(err, mf2.ErrSyntax):
		return fmt.Errorf("complex message: %w", err)
	case err != nil:
		resolutionErr = fmt.Errorf("complex message: %w", err)
	}

	switch b := message.ComplexBody.(type) {
	case ast.Matcher:
		err = e.resolveMatcher(b)
	case ast.QuotedPattern:
		err = e.resolvePattern(b)
	}

	if err != nil {
		return errors.Join(resolutionErr, fmt.Errorf("complex message: %w", err))
	}

	return resolutionErr
}

func (e *executer) resolveDeclarations(declarations []ast.Declaration) error { //nolint:unparam
	for _, decl := range declarations {
		switch d := decl.(type) {
		case ast.LocalDeclaration:
			r, err := e.resolveExpression(d.Expression)
			if err != nil {
				r.err = errors.Join(r.err, fmt.Errorf("resolve local %s: %w", d.Variable, err))
			}

			e.variables[string(d.Variable)] = r // newVariable(d.Expression, nil)
		case ast.InputDeclaration:
			r, err := e.resolveExpression(ast.Expression(d))
			if err != nil {
				r.err = errors.Join(r.err, fmt.Errorf("resolve input %s: %w", d.Operand, err))
			}

			e.variables[string(d.Operand.(ast.Variable))] = r //nolint: forcetypeassert // always ast.Variable
		}
	}

	return nil
}

func (e *executer) resolvePattern(pattern []ast.PatternPart) error {
	var resolutionErr error

	errorf := func(format string, args ...any) error {
		return errors.Join(resolutionErr, fmt.Errorf("pattern: "+format, args...))
	}

	for _, part := range pattern {
		switch v := part.(type) {
		case ast.Text:
			if _, err := e.w.Write([]byte(v)); err != nil {
				return errorf("write text: %w", err)
			}
		case ast.Expression:
			resolved, err := e.resolveExpression(v)
			if err != nil {
				resolutionErr = errors.Join(resolutionErr, err)
			}

			if _, err := e.w.Write([]byte(resolved.String())); err != nil {
				return errorf("write resolved expression: %w", err)
			}
		// When formatting to a string, markup placeholders format to an empty string by default.
		// See ".message-format-wg/exploration/open-close-placeholders.md#formatting-to-a-string"
		case ast.Markup:
		}
	}

	return resolutionErr
}

func (e *executer) resolveExpression(expr ast.Expression) (*ResolvedValue, error) {
	var (
		funcName      string
		options       Options
		resolutionErr error
	)

	value, err := e.resolveValue(expr.Operand)
	if err != nil {
		resolutionErr = errors.Join(resolutionErr, fmt.Errorf("expression: %w", err))
	}

	switch v := expr.Annotation.(type) {
	default:
		return newFallbackValue(expr), fmt.Errorf(`expression: %T annotation "%s": %w`, v, v, mf2.ErrUnsupportedExpression)
	case ast.Function:
		funcName = v.Identifier.String()

		if options, err = e.resolveOptions(v.Options); err != nil {
			return newFallbackValue(expr), fmt.Errorf("expression: %w", err)
		}
	case nil: // noop, no annotation
	}

	if funcName == "" {
		switch t := value.(type) {
		default:
			return newFallbackValue(expr), resolutionErr
		case *ResolvedValue: // the expression has already been resolved before
			return t, resolutionErr
		case string:
			funcName = "string"
		case float64:
			funcName = "number"
		}
	}

	f, ok := e.template.registry[funcName]
	if !ok {
		err = fmt.Errorf(`expression: %w "%s"`, mf2.ErrUnknownFunction, funcName)
		return newFallbackValue(expr), errors.Join(resolutionErr, err)
	}

	result, err := f(NewResolvedValue(value), options, e.template.locale)
	if err != nil {
		return newFallbackValue(expr), errors.Join(resolutionErr, fmt.Errorf("expression: %w", err))
	}

	return result, resolutionErr
}

// resolveValue resolves the value of an expression's operand.
//
//   - If the operand is a literal, it returns the literal's value.
//   - If the operand is a variable, it returns the value of the variable from the input map.
func (e *executer) resolveValue(v ast.Value) (any, error) {
	switch v := v.(type) {
	default: // this should never happen, should be cought by lexer/parser.
		return nil, fmt.Errorf(`unknown value type "%T"`, v)
	case nil:
		return v, nil // nil is also a valid value.
	case ast.QuotedLiteral:
		return string(v), nil
	case ast.NameLiteral:
		return string(v), nil
	case ast.NumberLiteral:
		return string(v), nil
	case ast.Variable:
		val, ok := e.variables[string(v)]
		if !ok {
			return NewResolvedValue("{" + v.String() + "}"), fmt.Errorf(`%w "%s"`, mf2.ErrUnresolvedVariable, v)
		}

		return val, val.err
	}
}

func (e *executer) resolveOptions(options []ast.Option) (Options, error) {
	m := make(Options, len(options))

	for _, opt := range options {
		value, err := e.resolveValue(opt.Value)
		if err != nil {
			return nil, fmt.Errorf("option: %w", err)
		}

		m[opt.Identifier.Name] = NewResolvedValue(value)
	}

	return m, nil
}

func (e *executer) resolveMatcher(m ast.Matcher) error {
	selectors, matcherErr := e.resolveSelectors(m)
	if matcherErr != nil && !errors.Is(matcherErr, mf2.ErrBadSelector) {
		return fmt.Errorf("matcher: %w", matcherErr)
	}

	pref := e.resolvePreferences(m, selectors)

	filteredVariants := e.filterVariants(m, pref)

	err := e.resolvePattern(e.bestMatchedPattern(filteredVariants, pref))
	if err != nil {
		return errors.Join(matcherErr, fmt.Errorf("matcher: %w", err))
	}

	return matcherErr
}

func (e *executer) resolveSelectors(m ast.Matcher) ([]*ResolvedValue, error) {
	var err error

	res := make([]*ResolvedValue, 0, len(m.Selectors))

	for _, selector := range m.Selectors {
		// Selector variable is ALWAYS resolved. Parser errors with ErrBadSelector
		// when selector has no annotation.
		v := e.variables[string(selector)]
		if v.selectKey == nil {
			err = errors.Join(err, fmt.Errorf(`%w "%s"`, mf2.ErrBadSelector, v))
		}

		if v.err != nil {
			err = errors.Join(err, fmt.Errorf("%w: %w", mf2.ErrBadSelector, v.err))
		}

		res = append(res, v)
	}

	return res, err
}

func (e *executer) resolvePreferences(m ast.Matcher, selectors []*ResolvedValue) [][]string {
	// Step 2: Resolve Preferences
	pref := make([][]string, 0, len(selectors))

	// all variants have the same number of keys
	n := len(m.Variants[0].Keys)

	for i := range selectors {
		keys := make([]string, 0, n)

		for _, variant := range m.Variants {
			// NOTE(mvilks): since collected keys will be compared to the selector,
			//	we need the keys's raw string value, not the representation of it
			//  e.g. the `1` should be equal to `|1|`
			var key string

			switch v := variant.Keys[i].(type) {
			case ast.CatchAllKey:
				continue
			case ast.QuotedLiteral:
				key = string(v)
			case ast.NameLiteral:
				key = string(v)
			case ast.NumberLiteral:
				key = v.String()
			}

			// add only unique keys
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}

		matches := matchSelectorKeys(selectors[i], keys)
		pref = append(pref, matches)
	}

	return pref
}

func (e *executer) filterVariants(m ast.Matcher, pref [][]string) []ast.Variant {
	// Step 3: Filter Variants
	var filteredVariants []ast.Variant

variantLoop:
	for _, variant := range m.Variants {
		for i, matchedSelectorKeys := range pref {
			// NOTE(mvilks): since collected keys will be compared to the selector,
			//	we need the keys's raw string value, not the representation of it
			//  e.g. the `1` should be equal to `|1|`
			_, ok := variant.Keys[i].(ast.CatchAllKey)
			if !ok && !slices.Contains(matchedSelectorKeys, keyString(variant.Keys[i])) {
				continue variantLoop
			}
		}

		filteredVariants = append(filteredVariants, variant)
	}

	return filteredVariants
}

func (e *executer) bestMatchedPattern(filteredVariants []ast.Variant, pref [][]string) ast.QuotedPattern {
	// Step 4: Sort Variants
	sortable := make([]sortableVariant, 0, len(filteredVariants))

	for _, variant := range filteredVariants {
		sortable = append(sortable, sortableVariant{Score: -1, Variant: variant})
	}

	for i := len(pref) - 1; i >= 0; i-- {
		matches := pref[i]

		for tupleIndex, tuple := range sortable {
			key := tuple.Variant.Keys[i]
			currentScore := len(matches)

			var ks string

			// NOTE(mvilks): since collected keys will be compared to the selector,
			//	we need the keys's raw string value, not the representation of it
			//  e.g. the `1` should be equal to `|1|`
			switch key := key.(type) {
			default:
				ks = keyString(key)
			case ast.CatchAllKey:
				sortable[tupleIndex].Score = currentScore
				continue
			}

			currentScore = slices.Index(matches, ks)

			sortable[tupleIndex].Score = currentScore
		}

		sort.Sort(sortableVariants(sortable))
	}

	return sortable[0].Variant.QuotedPattern
}

func keyString(key ast.VariantKey) string {
	switch k := key.(type) {
	default:
		return ""
	case ast.CatchAllKey:
		return "*"
	case ast.QuotedLiteral:
		return string(k)
	case ast.NameLiteral:
		return string(k)
	case ast.NumberLiteral:
		return string(k)
	}
}

func matchSelectorKeys(selector *ResolvedValue, keys []string) []string {
	if selector.selectKey == nil || selector.err != nil {
		return []string{"*"}
	}

	selected := selector.selectKey(keys)
	if selected == "" {
		return nil
	}

	return []string{selected}
}

type sortableVariant struct {
	Variant ast.Variant
	Score   int
}

type sortableVariants []sortableVariant

func (s sortableVariants) Len() int {
	return len(s)
}

func (s sortableVariants) Less(i, j int) bool {
	return s[i].Score < s[j].Score
}

func (s sortableVariants) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
