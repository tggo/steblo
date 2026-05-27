// Package steblo is a zero-dependency, rule-based stemmer for the Ukrainian
// language. It implements the Porter-style stemmer described in
// docs/algorithm.md. All functions are stateless and safe for concurrent use.
package steblo

import "unicode"

// Options controls normalisation and stemming behaviour. The zero value is NOT
// the default configuration; use DefaultOptions or Stem for sensible defaults.
type Options struct {
	// Strict applies the consonant-alternation cleanup after suffix stripping
	// (spec §5). Off by default. Over-strips by design (e.g. нога → но).
	Strict bool
	// Lowercase pre-lowercases the input via unicode.ToLower. Default true.
	Lowercase bool
	// NormalizeYo maps the Russian letters ё→е and ъ→ї for mixed corpora.
	// Default false.
	NormalizeYo bool
	// NormalizeApostr unifies apostrophe variants and removes them before RV
	// computation (spec §2). Default true.
	NormalizeApostr bool
}

// DefaultOptions returns the configuration used by Stem and StemRunes.
func DefaultOptions() Options {
	return Options{
		Strict:          false,
		Lowercase:       true,
		NormalizeYo:     false,
		NormalizeApostr: true,
	}
}

// Stem returns the rule-based stem of a Ukrainian word using DefaultOptions.
// Safe for concurrent use.
func Stem(word string) string {
	return StemWith(word, DefaultOptions())
}

// StemWith returns the stem of word under the given options.
func StemWith(word string, opts Options) string {
	if word == "" {
		return word
	}
	return string(StemRunesWith([]rune(word), opts))
}

// StemRunes is the allocation-conscious form of Stem. The returned slice may
// alias the input if no transformation is applied; callers that mutate the
// result must clone it first.
func StemRunes(word []rune) []rune {
	return StemRunesWith(word, DefaultOptions())
}

// StemRunesWith stems a rune slice under the given options. The returned slice
// may alias the input.
func StemRunesWith(word []rune, opts Options) []rune {
	w := normalize(word, opts)
	if len(w) == 0 {
		return word
	}

	// RV split (spec §1): find the first vowel; the prefix up to and including
	// it is preserved untouched, all stripping happens on the remainder.
	firstVowel := -1
	for i, r := range w {
		if isVowel(r) {
			firstVowel = i
			break
		}
	}
	if firstVowel == -1 {
		return w // no vowel: return unchanged (spec §6)
	}
	if firstVowel == len(w)-1 {
		return w // RV empty: nothing after the first vowel (spec §6)
	}

	rv := stemRV(w[firstVowel+1:], opts.Strict)

	// Every phase only truncates rv from the right (no insert, no re-slice of
	// the start), so the stem is always the left-anchored prefix of w ending
	// where rv now ends. Returning that sub-slice is allocation-free.
	return w[:firstVowel+1+len(rv)]
}

// normalize applies the Options-gated input transforms (spec §2). It returns a
// slice that may alias word when no transform fires. Decomposed (NFD) Cyrillic
// letters are always recomposed — see composeCyrillic — so input from sources
// that store NFD (notably macOS filenames) stems identically to NFC input.
func normalize(word []rune, opts Options) []rune {
	needsCopy := false
	for _, r := range word {
		if opts.Lowercase && r != unicode.ToLower(r) {
			needsCopy = true
			break
		}
		if opts.NormalizeApostr && isApostrophe(r) {
			needsCopy = true
			break
		}
		if opts.NormalizeYo && (r == 'ё' || r == 'Ё' || r == 'ъ' || r == 'Ъ') {
			needsCopy = true
			break
		}
		if isCyrillicCombiningMark(r) {
			needsCopy = true
			break
		}
	}
	if !needsCopy {
		return word
	}

	out := make([]rune, 0, len(word))
	for _, r := range word {
		// Recompose a decomposed Cyrillic letter: fold a combining mark into the
		// preceding base letter rather than emitting it as a separate rune.
		if isCyrillicCombiningMark(r) && len(out) > 0 {
			if composed, ok := composeCyrillic(out[len(out)-1], r); ok {
				out[len(out)-1] = applyYo(composed, opts)
				continue
			}
		}
		if opts.Lowercase {
			r = unicode.ToLower(r)
		}
		r = applyYo(r, opts)
		if opts.NormalizeApostr && isApostrophe(r) {
			continue // unify-and-delete: the reference removes apostrophes
		}
		out = append(out, r)
	}
	return out
}

func applyYo(r rune, opts Options) rune {
	if opts.NormalizeYo {
		switch r {
		case 'ё':
			r = 'е'
		case 'ъ':
			r = 'ї'
		}
	}
	return r
}

// isCyrillicCombiningMark reports whether r is one of the combining marks that
// appear in decomposed Ukrainian/Russian Cyrillic: combining breve (й) and
// combining diaeresis (ї, ё).
func isCyrillicCombiningMark(r rune) bool {
	return r == '̆' || r == '̈'
}

// composeCyrillic folds base+mark into a precomposed Cyrillic letter. base is
// expected lowercase (normalize lowercases before composing), but the uppercase
// forms are handled too for the Lowercase=false path.
func composeCyrillic(base, mark rune) (rune, bool) {
	switch mark {
	case '̆': // combining breve
		switch base {
		case 'и':
			return 'й', true
		case 'И':
			return 'Й', true
		}
	case '̈': // combining diaeresis
		switch base {
		case 'і':
			return 'ї', true
		case 'І':
			return 'Ї', true
		case 'е':
			return 'ё', true
		case 'Е':
			return 'Ё', true
		}
	}
	return 0, false
}

// isApostrophe matches the apostrophe variants unified by NormalizeApostr.
func isApostrophe(r rune) bool {
	switch r {
	case '\'', '`', 'ʼ', '‘', '’', '′':
		return true
	}
	return false
}
