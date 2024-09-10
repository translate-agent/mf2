package template

import (
	"fmt"
	"math"
	"strconv"

	"go.expect.digital/mf2"
	"golang.org/x/text/language"
)

// RegistryTestFunc is the implementation of the :test:function.
//
//nolint:gocognit
func RegistryTestFunc(name string) func(*ResolvedValue, Options, language.Tag) (*ResolvedValue, error) {
	if name != "format" && name != "select" {
		panic(`want "format" or "select" func name in ":test" namespace`)
	}

	return func(operand *ResolvedValue, options Options, _ language.Tag) (*ResolvedValue, error) {
		errorf := func(format string, args ...any) (*ResolvedValue, error) {
			return nil, fmt.Errorf("exec test:"+name+" function: "+format, args...)
		}

		v, err := parseNumberOperand(operand)
		if err != nil {
			return errorf("%w", mf2.ErrBadOperand)
		}

		opts, err := parseTestFunctionOptions(options)
		if err != nil {
			return errorf("%w", mf2.ErrBadOption)
		}

		switch opts.fails { //nolint:exhaustive
		case alwaysFail:
			return errorf("%w", mf2.ErrBadSelector)
		case formatFail:
			if name == "format" {
				return errorf("%w", mf2.ErrBadSelector)
			}
		case selectFail:
			if name == "select" {
				return errorf("%w", mf2.ErrBadSelector)
			}
		}

		format := func() string {
			var s string

			if v < 0 {
				s = "-"
			}

			s += strconv.Itoa(int(math.Floor(math.Abs(v))))

			if opts.decimalPlaces == 0 {
				return s
			}

			return s + "." + strconv.Itoa(int((math.Abs(v)-float64(int(math.Floor(math.Abs(v)))))*10)) //nolint:mnd
		}

		if name == "format" {
			return NewResolvedValue(v, WithFormat(format)), nil
		}

		selectKey := func(keys []string) string {
			if opts.fails == alwaysFail || opts.fails == selectFail {
				return ""
			}

			key := format()
			for _, k := range keys {
				if k == key {
					return k
				}
			}

			return ""
		}

		return NewResolvedValue(v, WithSelectKey(selectKey)), nil
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
