package template

import (
	"fmt"
	"maps"
	"math"
	"strconv"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

// RegistryTestFunc is the implementation of the ":test:function", ":test:format" and "test:select".
// ":test:function" is both functions - ":test:format" and ":test:select".
func RegistryTestFunc(name string) func(*ResolvedValue, Options, language.Tag) (*ResolvedValue, error) {
	isFormat := name == "format" || name == "function"
	isSelect := name == "select" || name == "function"

	if !isFormat && !isSelect {
		panic(`want "function", "format" or "select" func name in ":test" namespace`)
	}

	return func(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
		errorf := func(format string, args ...any) (*ResolvedValue, error) {
			return nil, fmt.Errorf("exec test:"+name+" function: "+format, args...)
		}

		v, err := parseNumberOperand(operand)
		if err != nil {
			return errorf("%w", mf2.ErrBadOperand)
		}

		// "If arg is the resolved value of an expression with a :test:function, :test:select, or :test:format
		// annotation for which resolution has succeeded, then [..]" merge all options from resolved value.
		switch operand.function {
		case ":test:function", ":test:format", ":test:select":
			merged := maps.Clone(operand.options)
			maps.Copy(merged, options)
			options = merged
		}

		opts, err := parseTestFunctionOptions(options)
		if err != nil {
			return errorf("%w", mf2.ErrBadOption)
		}

		switch opts.fails { //nolint:exhaustive
		case alwaysFail:
			return errorf("%w", mf2.ErrBadSelector)
		case formatFail:
			if isFormat {
				return errorf("%w", mf2.ErrBadSelector)
			}
		case selectFail:
			if isSelect {
				return errorf("%w", mf2.ErrBadSelector)
			}
		}

		format := func() string {
			// 1. If Input is less than 0, the character - U+002D Hyphen-Minus.
			// 2. The truncated absolute integer value of Input, i.e. floor(abs(Input)),
			//    formatted as a sequence of decimal digit characters (U+0030...U+0039).
			// 3. If DecimalPlaces is 1, then
			//   i.  The character . U+002E Full Stop.
			//   ii. The single decimal digit character representing the value floor((abs(Input) - floor(abs(Input))) * 10)
			if opts.decimalPlaces == 0 {
				return strconv.Itoa(int(v))
			}

			return fmt.Sprintf("%.1f", math.Trunc(v*10)/10) //nolint:mnd
		}

		selectKey := func(keys []string) string {
			key := format()
			for _, k := range keys {
				if k == key {
					return k
				}
			}

			return ""
		}

		withFunc := withFunction(":test:"+name, options)

		switch name {
		default: // :test:function
			return NewResolvedValue(v, WithFormat(format), WithSelectKey(selectKey), withFunc), nil
		case "format":
			return NewResolvedValue(v, WithFormat(format), withFunc), nil
		case "select":
			return NewResolvedValue(v, WithSelectKey(selectKey), withFunc), nil
		}
	}
}

type failsWhen string

const (
	neverFail  failsWhen = "never"
	selectFail failsWhen = "select"
	formatFail failsWhen = "format"
	alwaysFail failsWhen = "always"
)

type TestFunctionOptions struct {
	fails         failsWhen
	decimalPlaces int
}

func parseTestFunctionOptions(options Options) (TestFunctionOptions, error) {
	opts := TestFunctionOptions{fails: neverFail}

	for k, v := range options {
		switch k {
		default:
			// ignore any other option
		case "decimalPlaces":
			n, err := parseNumberOperand(v)
			if err != nil {
				return opts, fmt.Errorf("parse decimalPlaces operand: %v", v)
			}

			switch int(n) {
			default:
				return opts, fmt.Errorf("invalid decimalPlaces: %v", n)
			case 0, 1:
				opts.decimalPlaces = int(n)
			}
		case "fails":
			switch failsWhen(v.String()) {
			default:
				return opts, fmt.Errorf("invalid fails: %s", v)
			case neverFail, selectFail, formatFail, alwaysFail:
				opts.fails = failsWhen(v.String())
			}
		}
	}

	return opts, nil
}
