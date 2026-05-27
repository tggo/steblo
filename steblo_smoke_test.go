package steblo

import "testing"

// Smoke test against the spec §7 worked examples. This is a temporary check for
// Phase 2; the full corpus-driven tests arrive in Phase 4.
func TestSmokeSpecExamples(t *testing.T) {
	cases := []struct {
		word, want, wantStrict string
	}{
		{"випробування", "випробуван", "випробуван"},
		{"Чепинога", "чепиног", "чепино"},
		{"Чепинозі", "чепиноз", "чепино"},
		{"погашення", "погашен", "погашен"}, // D4 Snowball: нн undoubled
		{"тямущий", "тямущ", "тямущ"},
		{"рефлексивного", "рефлексивн", "рефлексивн"},
		{"нога", "ног", "но"},
	}
	for _, c := range cases {
		if got := Stem(c.word); got != c.want {
			t.Errorf("Stem(%q) = %q, want %q", c.word, got, c.want)
		}
		strict := StemWith(c.word, Options{Strict: true, Lowercase: true, NormalizeApostr: true})
		if strict != c.wantStrict {
			t.Errorf("StemWith(%q, strict) = %q, want %q", c.word, strict, c.wantStrict)
		}
	}
}

func TestNoVowel(t *testing.T) {
	if got := Stem("бгм"); got != "бгм" {
		t.Errorf("no-vowel word changed: %q", got)
	}
}
