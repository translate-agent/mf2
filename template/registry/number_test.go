package registry

import (
	"testing"

	"golang.org/x/text/language"
)

func Test_Number(t *testing.T) {
	t.Parallel()

	// decimal

	assert := assertFormat(t, numberRegistryFunc, nil, language.Latvian)
	assert(-0.1234, "-0,123")
	assert(0, "0")
	assert(0.1234, "0,123")

	assert = assertFormat(t, numberRegistryFunc, map[string]any{"signDisplay": "auto"}, language.AmericanEnglish)
	assert(-0.15, "-0.15")
	assert(0, "0")
	assert(0.15, "0.15")

	assert = assertFormat(t, numberRegistryFunc, map[string]any{"signDisplay": "always"}, language.AmericanEnglish)
	assert(-0.15, "-0.15")
	assert(0, "+0")
	assert(0.15, "+0.15")

	assert = assertFormat(t, numberRegistryFunc, map[string]any{"signDisplay": "exceptZero"}, language.AmericanEnglish)
	assert(-0.15, "-0.15")
	assert(0, "0")
	assert(0.15, "+0.15")

	assert = assertFormat(t, numberRegistryFunc, map[string]any{"signDisplay": "never"}, language.AmericanEnglish)
	assert(-0.15, "0.15")
	assert(0, "0")
	assert(0.15, "0.15")

	assert = assertFormat(t, numberRegistryFunc, map[string]any{"maximumFractionDigits": 1}, language.AmericanEnglish)
	assert(0.15, "0.1")

	// percent

	assert = assertFormat(t, numberRegistryFunc, map[string]any{"style": "percent"}, language.Latvian)
	assert(-0.127, "-13%")
	assert(0, "0%")
	assert(0.127, "13%")

	assert = assertFormat(t, numberRegistryFunc,
		map[string]any{"style": "percent", "signDisplay": "auto"}, language.AmericanEnglish)
	assert(-0.127, "-13%")
	assert(0, "0%")
	assert(0.127, "13%")

	assert = assertFormat(t, numberRegistryFunc,
		map[string]any{"style": "percent", "signDisplay": "always"}, language.AmericanEnglish)
	assert(-0.127, "-13%")
	assert(0, "+0%")
	assert(0.127, "+13%")

	assert = assertFormat(t, numberRegistryFunc,
		map[string]any{"style": "percent", "signDisplay": "exceptZero"}, language.AmericanEnglish)
	assert(-0.127, "-13%")
	assert(0, "0%")
	assert(0.127, "+13%")

	assert = assertFormat(t, numberRegistryFunc,
		map[string]any{"style": "percent", "signDisplay": "never"}, language.AmericanEnglish)
	assert(-0.127, "13%")
	assert(0, "0%")
	assert(0.127, "13%")

	assert = assertFormat(t, numberRegistryFunc,
		map[string]any{"style": "percent", "maximumFractionDigits": 1}, language.Latvian)
	assert(0.127, "12,7%")
}
