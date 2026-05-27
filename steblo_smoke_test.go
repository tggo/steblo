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

// TestNFDComposition: decomposed (NFD) Cyrillic — combining breve (й) and
// diaeresis (ї, ё) — must stem identically to precomposed (NFC) input. Real
// sources emit NFD (notably macOS filenames).
func TestNFDComposition(t *testing.T) {
	const br, di = rune(0x0306), rune(0x0308)
	cases := []struct {
		nfc string
		nfd []rune
	}{
		{"Київ", []rune{'к', 'и', 'і', di, 'в'}},
		{"гай", []rune{'г', 'а', 'и', br}},
		{"їжака", []rune{'і', di, 'ж', 'а', 'к', 'а'}},
		{"доброї", []rune{'д', 'о', 'б', 'р', 'о', 'і', di}},
	}
	for _, c := range cases {
		if a, b := Stem(c.nfc), Stem(string(c.nfd)); a != b {
			t.Errorf("NFC %q -> %q but NFD -> %q", c.nfc, a, b)
		}
	}
}

// TestNFDUppercase exercises composition on the Lowercase=false path, where the
// base letter retains its uppercase form (И/І/Е + combining mark).
func TestNFDUppercase(t *testing.T) {
	const br, di = rune(0x0306), rune(0x0308)
	opts := Options{Lowercase: false, NormalizeApostr: true}
	// uppercase И + breve must compose to Й (not stay decomposed).
	got := StemWith(string([]rune{'Г', 'А', 'И', br}), opts)
	if want := StemWith("ГАЙ", opts); got != want {
		t.Errorf("uppercase NFD %q != NFC %q", got, want)
	}
	// NormalizeYo applied to a composed ё (е + diaeresis → ё → е).
	yo := DefaultOptions()
	yo.NormalizeYo = true
	a := StemWith(string([]rune{'л', 'е', di, 'н'}), yo) // лён → composed then ё→е
	b := StemWith("лен", yo)
	if a != b {
		t.Errorf("NFD+NormalizeYo %q != %q", a, b)
	}
}

func TestNoVowel(t *testing.T) {
	if got := Stem("бгм"); got != "бгм" {
		t.Errorf("no-vowel word changed: %q", got)
	}
}
