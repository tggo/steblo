package steblo

// This file is the materialisation of docs/algorithm.md. Every table and phase
// here cites the spec section it implements. When in doubt, the spec wins.

// vowels — the ten-vowel set from spec §1. й is NOT a vowel here.
// ы appears only inside the perfective-gerund table as a legacy artifact.
func isVowel(r rune) bool {
	switch r {
	case 'а', 'е', 'и', 'о', 'у', 'ю', 'я', 'і', 'ї', 'є':
		return true
	}
	return false
}

func isConsonant(r rune) bool { return !isVowel(r) }

// Suffix tables (spec §4), longest-first within each group. Stored as [][]rune
// so matching is a plain tail comparison with no per-call allocation.
var (
	// §4.1 PERFECTIVE_GERUND — canonical = corrected (Node) form, divergence D1.
	pgUnconditional = toRunes("ившись", "ывшись", "ивши", "ывши", "ив", "ыв")
	// these strip only when preceded by а or я:
	pgAfterAYA = toRunes("вшись", "вши", "в")

	// §4.2 REFLEXIVE
	reflexive = toRunes("ся", "сь", "си")

	// §4.3 ADJECTIVE (divergence D2: lineage-root Python set)
	adjective = toRunes(
		"ими", "ова", "ове", "йми", "іми", "ого", "ому",
		"ій", "ий", "ів", "їй", "єє", "еє", "ім", "ем", "им", "их", "іх", "ою", "ої",
		"а", "е", "є", "я", "у", "ю",
	)

	// §4.4 PARTICIPLE (only after a successful ADJECTIVE strip)
	participle = toRunes(
		"ого", "ому", "йми",
		"ий", "им", "ім", "ій", "ою", "их",
		"а", "у", "і",
	)

	// §4.5 VERB
	verb = toRunes(
		"ать", "ять", "али", "учи", "ячи", "вши", "ати", "яти",
		"сь", "ся", "ив", "ав", "ши", "ме",
		"у", "ю", "е", "є",
	)

	// §4.6 NOUN (broadest, last-resort group)
	noun = toRunes(
		"ями", "ами", "иям", "ием", "иях", "ові", "еві",
		"ев", "ов", "еи", "ей", "ой", "ий", "ям", "ем", "ам", "ом", "ах", "ях",
		"ию", "ью", "ия", "ья", "ею", "єю", "ою", "єм", "ів", "їв",
		"а", "е", "и", "й", "о", "у", "ы", "ь", "ю", "я", "і", "ї", "є",
	)

	// §5 strict-mode terminal consonant-alternation clusters (divergence D5,
	// Node-only). Single-consonant members of [гджзкстхцчш] are handled after
	// the multi-rune clusters below.
	alterations = toRunes(
		"ждж",
		"ст", "дж", "ьц", "сі", "ці", "зі", "он", "ін", "ів", "ев", "ок", "шк",
	)
	alterationSingles = "гджзкстхцчш"
)

func toRunes(ss ...string) [][]rune {
	out := make([][]rune, len(ss))
	for i, s := range ss {
		out[i] = []rune(s)
	}
	return out
}

// endsWith reports whether rv ends with suf.
func endsWith(rv, suf []rune) bool {
	if len(suf) > len(rv) {
		return false
	}
	off := len(rv) - len(suf)
	for i, r := range suf {
		if rv[off+i] != r {
			return false
		}
	}
	return true
}

// stripFirst removes the first matching suffix from the group (the group is
// longest-first, so this is a longest-match strip). Reports whether it changed.
func stripFirst(rv []rune, group [][]rune) ([]rune, bool) {
	for _, suf := range group {
		if endsWith(rv, suf) {
			return rv[:len(rv)-len(suf)], true
		}
	}
	return rv, false
}

// stemRV applies all four phases to the post-first-vowel region, per spec §3.
// It mutates and returns the (re-sliced) rv.
func stemRV(rv []rune, strict bool) []rune {
	// ---- Step 1 ----
	if r, ok := stripPerfectiveGerund(rv); ok {
		rv = r // exclusive: a perfective-gerund strip short-circuits the block
	} else {
		rv, _ = stripFirst(rv, reflexive) // non-exclusive

		if r, ok := stripFirst(rv, adjective); ok {
			rv = r
			rv, _ = stripFirst(rv, participle)
		} else if r, ok := stripFirst(rv, verb); ok {
			rv = r
		} else {
			rv, _ = stripFirst(rv, noun)
		}
	}

	// ---- Step 2: terminal и ---- (spec §4.7)
	if n := len(rv); n > 0 && rv[n-1] == 'и' {
		rv = rv[:n-1]
	}

	// ---- Step 3: DERIVATIONAL → ость ---- (spec §4.8)
	if hasDerivationalShape(rv) {
		if endsWith(rv, []rune("ость")) {
			rv = rv[:len(rv)-4] // remove the 4 runes о с т ь (bare ост is a no-op, D3)
		}
	}

	// ---- Step 4: Snowball-Russian, three ordered exclusive branches ---- (§4.9, D4)
	rv = stepFourSnowball(rv)

	// ---- strict mode (spec §5) ----
	if strict {
		rv = stripAlterations(rv)
	}
	return rv
}

// stripPerfectiveGerund implements spec §4.1 / D1.
func stripPerfectiveGerund(rv []rune) ([]rune, bool) {
	if r, ok := stripFirst(rv, pgUnconditional); ok {
		return r, true
	}
	for _, suf := range pgAfterAYA {
		if endsWith(rv, suf) {
			before := len(rv) - len(suf)
			if before > 0 && (rv[before-1] == 'а' || rv[before-1] == 'я') {
				return rv[:before], true
			}
		}
	}
	return rv, false
}

// hasDerivationalShape reports whether rv matches the spec §4.8 gate:
// it ends in ост or ость, and somewhere contains the shape C V+ C+ V.
func hasDerivationalShape(rv []rune) bool {
	if !endsWith(rv, []rune("ост")) && !endsWith(rv, []rune("ость")) {
		return false
	}
	for i := 0; i < len(rv); i++ {
		j := i
		if !isConsonant(rv[j]) {
			continue
		}
		j++
		n := 0
		for j < len(rv) && isVowel(rv[j]) {
			j++
			n++
		}
		if n == 0 {
			continue
		}
		n = 0
		for j < len(rv) && isConsonant(rv[j]) {
			j++
			n++
		}
		if n == 0 {
			continue
		}
		if j < len(rv) && isVowel(rv[j]) {
			return true // remaining .* is unconstrained
		}
	}
	return false
}

// stepFourSnowball implements spec §4.9 (divergence D4): one of three
// mutually-exclusive branches, tried in order.
func stepFourSnowball(rv []rune) []rune {
	// (1) undouble нн
	if endsWith(rv, []rune("нн")) {
		return rv[:len(rv)-1]
	}
	// (2) superlative ейш / ейше, then undouble нн
	if endsWith(rv, []rune("ейше")) {
		rv = rv[:len(rv)-4]
		return undoubleNN(rv)
	}
	if endsWith(rv, []rune("ейш")) {
		rv = rv[:len(rv)-3]
		return undoubleNN(rv)
	}
	// (3) remove terminal ь
	if n := len(rv); n > 0 && rv[n-1] == 'ь' {
		return rv[:n-1]
	}
	return rv
}

func undoubleNN(rv []rune) []rune {
	if endsWith(rv, []rune("нн")) {
		return rv[:len(rv)-1]
	}
	return rv
}

// stripAlterations implements spec §5 strict-mode terminal cluster removal.
func stripAlterations(rv []rune) []rune {
	if r, ok := stripFirst(rv, alterations); ok {
		return r
	}
	if n := len(rv); n > 0 {
		for _, c := range alterationSingles {
			if rv[n-1] == c {
				return rv[:n-1]
			}
		}
	}
	return rv
}
