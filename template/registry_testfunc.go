package template

import (
	"fmt"
	"strconv"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

// RegistryTestFunc is the implementation of the :test:function.
func RegistryTestFunc(usage string) func(*ResolvedValue, Options, language.Tag) (*ResolvedValue, error) {
	return func(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
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

		switch opts.fails {
		default:
			// noop
		case alwaysFail:
			return errorf("bad operand: %w", mf2.ErrBadOperand)
		case formatFail:
			if usage == "format" {
				return errorf("bad operand: %w", mf2.ErrBadOperand)
			}
		case selectFail:
			if usage == "select" {
				return errorf("bad operand: %w", mf2.ErrBadOperand)
			}
		}

		f := func() string {
			var s string

			if v < 0 {
				s = "-"
			}

			s += strconv.Itoa(int(v))

			if opts.decimalPlaces == 0 {
				return s
			}

			return s + "." + strconv.Itoa(int((v-float64(int(v)))*10)) //nolint:mnd
		}

		s := func(keys []string) string {
			if opts.fails == alwaysFail || opts.fails == selectFail {
				return ""
			}

			for _, k := range keys {
				if k == f() {
					return k
				}
			}

			return ""
		}

		return NewResolvedValue(v, WithSelectKey(s), WithFormat(f)), nil
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
