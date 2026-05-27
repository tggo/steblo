package steblo

import (
	"testing"
	"unicode/utf8"
)

var fuzzSeeds = []string{
	"випробування", "Чепинога", "погашення", "з'ясування", "найдобрейший",
	"", "а", "ь", "нн", "ейше", "ость", "прочитавши", "бгм", "АБВ",
}

// FuzzStemNoCrash is a crash-only oracle: Stem must never panic and must always
// return valid UTF-8.
func FuzzStemNoCrash(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := Stem(in)
		if !utf8.ValidString(out) {
			t.Fatalf("Stem(%q) returned invalid UTF-8: %q", in, out)
		}
	})
}

// FuzzStemLengthMonotonic: the stem is never longer (in runes) than the input.
// (Idempotence is NOT a property of this algorithm — see docs/algorithm.md §9a.)
func FuzzStemLengthMonotonic(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		if out, n := Stem(in), len([]rune(in)); len([]rune(out)) > n {
			t.Fatalf("Stem(%q)=%q grew from %d runes", in, out, n)
		}
	})
}
