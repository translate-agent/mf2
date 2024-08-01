package template

import (
	"fmt"

	"golang.org/x/text/language"
)

// integerRegistryFunc is the implementation of the integer function. Locale-sensitive integer formatting.
var integerRegistryFunc = RegistryFunc{
	Format: integerFunc,
	Select: integerFunc,
}

func integerFunc(operand any, options Options, locale language.Tag) (any, error) {
	if options == nil {
		options = Options{"maximumFractionDigits": 0}
	} else {
		options["maximumFractionDigits"] = 0
	}

	value, err := numberFunc(operand, options, locale)
	if err != nil {
		return nil, fmt.Errorf("exec integer func: %w", err)
	}

	return value, nil
}
