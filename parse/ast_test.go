package parse

import "testing"

func BenchmarkComplexMessage_String(b *testing.B) {
	//nolint:dupword
	tree, err := Parse(".match {$foo :number} {$bar :number} one one {{one one}} one * {{one other}} * * {{other}}")
	if err != nil {
		b.Error(err)
	}

	var result string

	for range b.N {
		result = tree.String()
	}

	_ = result
}
