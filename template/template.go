package template

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
	"golang.org/x/text/language"

	"go.expect.digital/mf2"
	ast "go.expect.digital/mf2/parse"
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
	ast      *ast.AST
	registry Registry
	locale   language.Tag
}

type variable struct {
	name        *string
	formatValue *string // TODO(mvilks): "selectValue" to be implemented
	expression  ast.Expression
}

func newVariable(e ast.Expression, name *string) variable { return variable{expression: e, name: name} }

func (v variable) Format(e *executer) (string, error) {
	if v.formatValue != nil {
		return *v.formatValue, nil
	}

	val, err := e.resolveExpression(v.expression)
	if err != nil && v.name != nil {
		n := "{$" + *v.name + "}"
		val = n
	}

	v.formatValue = &val

	return *v.formatValue, err
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
func WithFunc(name string, f RegistryFunc) Option {
	return func(t *Template) {
		t.registry[name] = f
	}
}

// WithFuncs adds functions to function registry.
func WithFuncs(reg Registry) Option {
	return func(t *Template) {
		for k, f := range reg {
			t.registry[k] = f
		}
	}
}

func WithLocale(locale language.Tag) Option {
	return func(t *Template) {
		t.locale = locale
	}
}

// Parse parses the MessageFormat2 string and returns the template.
func (t *Template) Parse(input string) (*Template, error) {
	ast, err := ast.Parse(input)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	t.ast = &ast

	return t, nil
}

// Execute writes the result of the template to the given writer.
func (t *Template) Execute(w io.Writer, input map[string]any) error {
	if t.ast == nil {
		return errors.New("execute template: AST is nil")
	}

	executer := &executer{template: t, w: w, variables: make(map[string]any, len(input))}

	for k, v := range input {
		executer.variables[k] = v
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
	variables map[string]any
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
	case errors.Is(err, mf2.ErrUnsupportedStatement),
		errors.Is(err, mf2.ErrUnresolvedVariable),
		errors.Is(err, mf2.ErrBadOperand),
		errors.Is(err, mf2.ErrBadOption):
		resolutionErr = fmt.Errorf("complex message: %w", err)
	case err != nil:
		return fmt.Errorf("complex message: %w", err)
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

func newLiteral(v any) ast.Value { //nolint:ireturn
	switch t := v.(type) {
	case string:
		return ast.QuotedLiteral(t)
	case float64:
		return ast.NumberLiteral(t)
	case int:
		return ast.NumberLiteral(t)
	}

	return nil
}

func (e *executer) resolveDeclarations(declarations []ast.Declaration) error {
	for _, decl := range declarations {
		switch d := decl.(type) {
		case ast.ReservedStatement:
			return fmt.Errorf("%w", mf2.ErrUnsupportedStatement)
		case ast.LocalDeclaration:
			e.variables[string(d.Variable)] = newVariable(d.Expression, nil)
		case ast.InputDeclaration:
			name := string(d.Operand.(ast.Variable)) //nolint: forcetypeassert // always ast.Variable

			val, ok := e.variables[name]
			if !ok {
				return mf2.ErrUnresolvedVariable
			}

			expr := ast.Expression(d)
			expr.Operand = newLiteral(val)
			e.variables[name] = newVariable(expr, &name)
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

			if _, err := e.w.Write([]byte(resolved)); err != nil {
				return errorf("write resolved expression: %w", err)
			}
		// When formatting to a string, markup placeholders format to an empty string by default.
		// See ".message-format-wg/exploration/open-close-placeholders.md#formatting-to-a-string"
		case ast.Markup:
		}
	}

	return resolutionErr
}

func (e *executer) resolveExpression(expr ast.Expression) (string, error) {
	value, err := e.resolveValue(expr.Operand)
	if err != nil {
		return fmt.Sprint(value), fmt.Errorf("expression: %w", err)
	}

	var (
		funcName      string
		options       map[string]any
		resolutionErr error
	)

	switch v := expr.Annotation.(type) {
	default:
		return "", fmt.Errorf(`expression: %T annotation "%s": %w`, v, v, mf2.ErrUnsupportedExpression)
	case ast.Function:
		funcName = v.Identifier.Name

		if options, err = e.resolveOptions(v.Options); err != nil {
			return "", fmt.Errorf("expression: %w", err)
		}
	case ast.PrivateUseAnnotation:
		// See ".message-format-wg/spec/formatting.md".
		//
		// Supported private-use annotation with no operand: the annotation starting sigil, optionally followed by
		// implementation-defined details conforming with patterns in the other cases (such as quoting literals).
		// If details are provided, they SHOULD NOT leak potentially private information.
		resolutionErr = fmt.Errorf(`expression: private use annotation "%s": %w`, v, mf2.ErrUnsupportedExpression)

		if value == nil {
			return "{" + string(v.Start) + "}", resolutionErr
		}
	case ast.ReservedAnnotation:
		resolutionErr = fmt.Errorf(`expression: reserved annotation "%s": %w`, v, mf2.ErrUnsupportedExpression)

		if value == nil {
			return "{" + string(v.Start) + "}", resolutionErr
		}
	case nil: // noop, no annotation
	}

	fmtErroredExpr := func() string {
		wrap := func(s fmt.Stringer) string { return "{" + s.String() + "}" }

		switch v := expr.Operand.(type) {
		default:
			return wrap(expr.Annotation)
		case ast.Variable:
			return wrap(v)
		case ast.NameLiteral, ast.NumberLiteral:
			return wrap(ast.QuotedLiteral(v.String()))
		case ast.QuotedLiteral:
			return wrap(v)
		}
	}

	if funcName == "" {
		switch t := value.(type) {
		default: // TODO(jhorsts): how is unknown type formatted?
			return fmtErroredExpr(), resolutionErr
		case variable:
			f, err := t.Format(e) //nolint:govet

			switch {
			case errors.Is(err, mf2.ErrUnresolvedVariable),
				errors.Is(err, mf2.ErrBadOperand),
				errors.Is(err, mf2.ErrBadOption):
				return f, err
			case err != nil:
				return fmtErroredExpr(), errors.Join(resolutionErr, err)
			}

			return f, nil
		case string:
			funcName = "string"
		case float64:
			funcName = "number"
		}
	}

	f, ok := e.template.registry[funcName] // TODO(jhorsts): lookup by namespace and name
	if !ok {
		err = fmt.Errorf(`expression: %w "%s"`, mf2.ErrUnknownFunction, funcName)
		return fmtErroredExpr(), errors.Join(resolutionErr, err)
	}

	if f.Format == nil {
		return "", fmt.Errorf(`expression: function "%s" not allowed in formatting context`, funcName)
	}

	result, err := f.Format(value, options, e.template.locale)
	if err != nil {
		return fmtErroredExpr(), errors.Join(resolutionErr, fmt.Errorf("expression: %w", err))
	}

	return fmt.Sprint(result), resolutionErr
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
		return float64(v), nil
	case ast.Variable:
		val, ok := e.variables[string(v)]
		if !ok {
			return "{" + v.String() + "}", fmt.Errorf(`%w "%s"`, mf2.ErrUnresolvedVariable, v)
		}

		return val, nil
	}
}

func (e *executer) resolveOptions(options []ast.Option) (map[string]any, error) {
	m := make(map[string]any, len(options))

	for _, opt := range options {
		name := opt.Identifier.Name
		if _, ok := m[name]; ok {
			return nil, fmt.Errorf(`%w "%s"`, mf2.ErrDuplicateOptionName, name)
		}

		value, err := e.resolveValue(opt.Value)
		if err != nil {
			return nil, fmt.Errorf("option: %w", err)
		}

		m[name] = value
	}

	return m, nil
}

func (e *executer) resolveMatcher(m ast.Matcher) error {
	res, matcherErr := e.resolveSelector(m)

	switch {
	case errors.Is(matcherErr, mf2.ErrUnknownFunction),
		errors.Is(matcherErr, mf2.ErrUnresolvedVariable): // noop
	case matcherErr != nil:
		return fmt.Errorf("matcher: %w", matcherErr)
	}

	pref := e.resolvePreferences(m, res)

	filteredVariants := e.filterVariants(m, pref)

	err := e.resolvePattern(e.bestMatchedPattern(filteredVariants, pref))
	if err != nil {
		return errors.Join(matcherErr, fmt.Errorf("matcher: %w", err))
	}

	return matcherErr
}

func (e *executer) resolveSelector(matcher ast.Matcher) ([]any, error) {
	var selectorErr error

	selectors := make([]any, 0, len(matcher.Selectors))

	addErr := func(err error) {
		selectorErr = errors.Join(selectorErr, fmt.Errorf("selector: %w", err))

		selectors = append(selectors, ast.CatchAllKey{})
	}

	for _, selector := range matcher.Selectors {
		var function ast.Function

		switch annotation := selector.Annotation.(type) {
		case nil:
			return nil, mf2.ErrMissingSelectorAnnotation
		case ast.ReservedAnnotation, ast.PrivateUseAnnotation:
			return nil, mf2.ErrUnsupportedExpression
		case ast.Function:
			function = annotation
		}

		f, ok := e.template.registry[function.Identifier.Name]
		if !ok {
			addErr(fmt.Errorf(`%w "%s"`, mf2.ErrUnknownFunction, function.Identifier.Name))
			continue
		}

		// TODO(jhorsts): what is match and format context? Does MF2 still have it?
		if f.Select == nil {
			return nil, fmt.Errorf(`selector: function "%s" not allowed`, function.Identifier.Name)
		}

		opts, err := e.resolveOptions(function.Options)
		if err != nil {
			addErr(err)
			continue
		}

		input, err := e.resolveValue(selector.Operand)
		if err != nil {
			addErr(err)
			continue
		}

		if t, ok := input.(variable); ok {
			input, _ = t.Format(e)
		}

		rslt, err := f.Select(input, opts, e.template.locale)
		if err != nil {
			addErr(err)
			continue
		}

		selectors = append(selectors, rslt)
	}

	return selectors, selectorErr
}

func (e *executer) resolvePreferences(m ast.Matcher, res []any) [][]string {
	// Step 2: Resolve Preferences
	pref := make([][]string, 0, len(res))

	for i := range res {
		var keys []string

		for _, variant := range m.Variants {
			for _, vKey := range variant.Keys {
				// NOTE(mvilks): since collected keys will be compared to the selector,
				//	we need the keys's raw string value, not the representation of it
				//  e.g. the `1` should be equal to `|1|`
				switch key := vKey.(type) {
				case ast.CatchAllKey:
					continue
				case ast.QuotedLiteral:
					keys = append(keys, string(key))
				case ast.NameLiteral:
					keys = append(keys, string(key))
				case ast.NumberLiteral:
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

			var ks string

			// NOTE(mvilks): since collected keys will be compared to the selector,
			//	we need the keys's raw string value, not the representation of it
			//  e.g. the `1` should be equal to `|1|`
			switch key := key.(type) {
			case ast.CatchAllKey:
				continue
			case ast.QuotedLiteral:
				ks = string(key)
			case ast.NameLiteral:
				ks = string(key)
			case ast.NumberLiteral:
				ks = key.String()
			}

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

func (e *executer) bestMatchedPattern(filteredVariants []ast.Variant, pref [][]string) ast.QuotedPattern {
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

			var ks string

			// NOTE(mvilks): since collected keys will be compared to the selector,
			//	we need the keys's raw string value, not the representation of it
			//  e.g. the `1` should be equal to `|1|`
			switch key := key.(type) {
			case ast.CatchAllKey:
				sortable[tupleIndex].Score = currentScore
				continue
			case ast.QuotedLiteral:
				ks = string(key)
			case ast.NameLiteral:
				ks = string(key)
			case ast.NumberLiteral:
				ks = key.String()
			}

			currentScore = slices.Index(matches, ks)

			sortable[tupleIndex].Score = currentScore
		}

		sort.Sort(SortableVariants(sortable))
	}

	return sortable[0].Variant.QuotedPattern
}

func matchSelectorKeys(rv any, keys []string) []string {
	value, ok := rv.(string)
	if !ok {
		return nil
	}

	var matches []string

	for _, key := range keys {
		if key == value {
			matches = append(matches, key)
		}
	}

	return matches
}

type SortableVariant struct {
	Variant ast.Variant
	Score   int
}

type SortableVariants []SortableVariant

func (s SortableVariants) Len() int {
	return len(s)
}

func (s SortableVariants) Less(i, j int) bool {
	return s[i].Score < s[j].Score
}

func (s SortableVariants) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
