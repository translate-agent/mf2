package template

import (
	"fmt"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

// RegistryTestFunc is the implementation of the :test:function.
func RegistryTestFunc(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
	errorf := func(format string, args ...any) (*ResolvedValue, error) {
		return nil, fmt.Errorf("exec string function: "+format, args...)
	}

	v, err := parseNumberOperand(operand)
	if err != nil {
		return errorf("bad operand: %w", mf2.ErrBadOperand)
	}

	opts, err := parseTestFunctionOptions(options)
	if err != nil {
		return errorf("bad option: %w", mf2.ErrBadOption)
	}

	_ = v
	_ = opts

	return operand, nil
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
				return opts, fmt.Errorf("invalid value for decimalPlaces: %v", n)
			case 0, 1:
				opts.decimalPlaces = int(n)
			}
		case "fails":
			switch v.String() {
			default:
				return opts, fmt.Errorf("bad value given for fails: %s", v)
			case string(neverFail), string(selectFail), string(formatFail), string(alwaysFail):
				opts.fails = failsWhen(v.String())
			}
		}
	}

	return opts, nil
}
