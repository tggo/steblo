# steblo — canonical algorithm specification

This document is the load-bearing artifact of the project. The Go code in
`steblo.go` is one materialisation of the rules defined here. When code and spec
disagree, the spec is authoritative — fix the code or amend the spec with a
written rationale; never let them drift silently.

## Lineage

steblo implements a Porter-style rule-based Ukrainian truncation stemmer derived
from the classic Snowball-Russian algorithm. The behaviour was reconstructed
from several independent open-source implementations of the same algorithm:

- an early **Python** implementation and a later, groomed **Python** port — rule
  logic byte-for-byte equivalent, carrying a passing self-test;
- a **Node.js** port;
- a second, independent **JS** port.

These are referred to throughout as the *Node reference*, the *Python
reference(s)*, and the *second JS port*. The original PHP module the lineage
descends from is no longer reachable. "Canonical" is anchored on the Python
references, cross-checked against the JS ports; every divergence is resolved with
a written rationale (the divergence ledger, §8).

Local copies used to reconstruct the spec live under `references/` (gitignored,
not distributed).

---

## 1. Alphabet and regions

### Vowel set

```
а е и о у ю я і ї є
```

Ten vowels. All references use exactly this set
(`[аеиоуюяіїє]`). `й` is **not** a vowel here (it is treated as a consonant /
semivowel). Russian `ы`, `э`, `ё` are not in the set; `ы` appears only inside
the perfective-gerund pattern as a legacy artifact (see §4.1).

### RV region — the only region used

Despite the project brief mentioning R1/R2, the upstream algorithm uses a
**single region, RV**, exactly as Snowball defines it for Russian:

> **RV** is the part of the word **after the first vowel**. If the word has no
> vowel, RV is empty (and the word is returned unchanged).

Concretely, split the word into:

- `prefix` = `word[0 .. i]` inclusive, where `i` is the index of the first vowel.
- `RV`     = `word[i+1 ..]` (everything after the first vowel).

All suffix stripping operates **only on RV**. The final stem is
`prefix + RV_after_stripping`. This is why short words keep their head: e.g. the
first vowel and everything before it is never touched.

There is **no R2**. The `DERIVATIONAL` test (§4.8) substitutes for what R2 would
gate in classic Porter, by requiring a specific consonant/vowel shape inside RV.

---

## 2. Input preprocessing (normalisation)

The Python reference does, unconditionally:

1. lowercase
2. delete apostrophes (`'`)
3. `ё → е`
4. `ъ → ї`

The Node port does **lowercase only**. the second JS port does **lowercase only**.

### steblo decision

Map this onto `Options` (see `steblo.go` API), defaults chosen to match the
*most common modern Ukrainian* use, not the legacy Russian-letter handling:

| Step | steblo default | Option |
|---|---|---|
| lowercase | on | `Lowercase` (default true) |
| apostrophe unify/strip | on | `NormalizeApostr` (default true) |
| `ё → е` | **off** | `NormalizeYo` (default false) |
| `ъ → ї` | folded into `NormalizeYo` | — |
| NFD → NFC recompose | **always** | — (correctness, not optional) |

steblo additionally recomposes decomposed (NFD) Cyrillic letters that none of
the references handle: combining breve (`и` + ◌̆ → `й`) and combining diaeresis
(`і` + ◌̈ → `ї`, `е` + ◌̈ → `ё`). This is unconditional — decomposed input (e.g.
text read from macOS filenames, which are NFD) otherwise fails every suffix and
vowel match and would silently not stem. It only affects decomposed input and
never alters NFC text.

Rationale: apostrophe handling is needed for correct Ukrainian RV computation
(`п'ять`, `об'єкт`). `ё`/`ъ` are Russian letters; defaulting their rewrite off
keeps clean Ukrainian text untouched, but the option exists for mixed corpora to
reproduce the reference exactly. Apostrophe handling: the reference *deletes* the
apostrophe; steblo with `NormalizeApostr` first unifies `ʼ ' \` ′` → U+02BC then
deletes it before RV computation, matching reference behaviour on the resulting
letters.

---

## 3. Phase order (top level)

Operating on RV, in this exact order:

```
Step 1  PERFECTIVE_GERUND   — if it strips, skip the whole REFLEXIVE/…/NOUN block
        else:
          REFLEXIVE          (always attempted)
          ADJECTIVE          — if it strips:
              PARTICIPLE     (attempted once, after adjective)
          else:
              VERB           — if it strips: done with this block
              else:
                  NOUN
Step 2  remove terminal и
Step 3  if DERIVATIONAL shape matches: remove terminal ость
Step 4  (Snowball, three mutually-exclusive branches, tried in order):
          (1) if ends нн      → undouble: нн → н        (done)
          (2) else if ends SUPERLATIVE ейш(е)
                              → remove it, then undouble нн → н   (done)
          (3) else if ends ь  → remove ь
```

Key control-flow facts (verified against all references unless noted):

- **Step 1 is exclusive at the top**: a successful PERFECTIVE_GERUND strip
  short-circuits past REFLEXIVE/ADJECTIVE/VERB/NOUN entirely.
- **REFLEXIVE is not exclusive**: it is attempted, then ADJECTIVE branch runs
  regardless of whether REFLEXIVE matched.
- **At most one of {ADJECTIVE-then-PARTICIPLE} vs {VERB} vs {NOUN}** fires:
  ADJECTIVE wins if it matches (and only then is PARTICIPLE attempted); else VERB;
  else NOUN.
- Each step strips **at most one** suffix (single `re.sub`/`replace`, longest
  match — see §4).
- "if it strips" means the string actually changed.

### Longest-match semantics

Reference impls express each suffix group as a single regex alternation anchored
at `$`. Because regex search is leftmost-first and the match must reach end of
string, the *longest* applicable suffix is the one removed (its match starts at
the earliest index). steblo reproduces this with **sorted longest-first suffix
lists** scanned in order; first hit wins. This is behaviourally identical to the
reference regexes for all suffix groups.

---

## 4. Suffix groups (canonical, longest-first)

Canonical lists are taken from the Python reference. Where the Node / second-JS-port
sets differ, it is called out under **Divergence**. Suffixes are given
longest-first; the Go tables must preserve this order.

### 4.1 PERFECTIVE_GERUND (Step 1) — **CANONICAL = corrected (Node) form**

Intended suffixes:

- unconditional: `ившись`, `ивши`, `ів`(no) … precisely: `ив`, `ивши`, `ившись`,
  `ыв`, `ывши`, `ывшись`
- **only after а or я**: `вшись`, `вши`, `в`

Reference regexes:

- Node: `(?:[иы]в(?:ши(?:сь)?)?|(?<=[ая])(?:в(?:ши(?:сь)?)?))$` — correct.
- Python (both): `(ив|ивши|ившись|ыв|ывши|ывшись((?<=[ая])(в|вши|вшись)))$` —
  **buggy**: the `(?<=[ая])(в|вши|вшись)` group is mis-nested as a *suffix of the
  `ывшись` alternative*, so the а/я-conditioned branch is effectively dead. Only
  `ив|ивши|ившись|ыв|ывши|ывшись` ever match.
- the second JS port: `((ив|ивши|ившись|ыв|ывши|ывшись(в|вши|вшись)))$` — lookbehind
  dropped entirely; the trailing `(в|вши|вшись)` is likewise unreachable.

> **DIVERGENCE D1 — perfective gerund.** Canonical adopts the **Node (corrected)**
> form. Rationale: the lookbehind is clearly *intended* (it is present in the
> original, merely mis-nested) and matches Snowball-Russian's `perfective_gerund`
> group 2 (`в вши вшись` preceded by а/я). The Python/the second JS port forms contain an
> unintentional regex bug that disables a whole linguistic branch. Consequence:
> for words like `прочитавши`, `зробивши`-after-я forms, steblo strips the
> gerund where the Python and JS ports leave it. These are recorded as `consensus:false`
> rows with this rationale.

steblo realisation: match (longest-first) `ившись, ивши, ив, ывшись, ывши, ыв`
unconditionally; then `вшись, вши, в` **only if** the rune immediately before the
matched suffix is `а` or `я`.

### 4.2 REFLEXIVE (Step 1, non-exclusive)

`с[яьи]$` → suffixes `ся`, `сь`, `си`. All references identical. No divergence.

### 4.3 ADJECTIVE (Step 1)

Canonical (Python), longest-first deduped:

```
ими  іми  йми  ова  ове  ого  ому  ої
ій  ий  ів  єє  еє  їй  ім  ем  им  их  іх  ою
а  е  є  я  у  ю
```

> **DIVERGENCE D2 — adjective set.** Node uses a different alternation that adds
> `еє ем`-style forms and several и/і pairs and omits `ова ове`. The sets overlap
> heavily but are not equal. Canonical follows the Python-reference set above.
> Affected words recorded in corpus where outputs differ.

### 4.4 PARTICIPLE (Step 1, only after a successful ADJECTIVE strip)

Canonical (Python), longest-first deduped:

```
ого  ому  йми
ий  им  ім  ій  ою  их
а  у  і
```

### 4.5 VERB (Step 1)

Canonical (Python), longest-first deduped:

```
учи  ячи  ати  яти  али
ать  ять  вши  ши  ив  ме  сь  ся  ав
е  є  у  ю
```

### 4.6 NOUN (Step 1)

Canonical (Python), longest-first deduped:

```
ями  ами  иям  иях  ією(no)
ові  еві  ием  єю  ею  ою  ью  ия  ья  ів  їв  иям  ям  ях
ев  ов  еи  ей  ой  ий  ам  ом  ию  ем  єм  ам
а  е  и  й  о  у  ы  ь  ю  я  і  ї  є
```

(The Go table will carry the exact deduped set; the grouping above is for
reading. Source alternation:
`а|ев|ов|е|ями|ами|еи|и|ей|ой|ий|й|иям|ям|ием|ем|ам|ом|о|у|ах|иях|ях|ы|ь|ию|ью|ю|ия|ья|я|і|ові|ї|ею|єю|ою|є|еві|ем|єм|ів|їв|ю`,
deduped, sorted longest-first.)

> Note: the NOUN set is the broadest and is the last-resort group; many forms it
> contains overlap with single-vowel endings already covered by ADJECTIVE/VERB,
> but NOUN only runs when neither ADJECTIVE nor VERB fired.

### 4.7 Step 2 — terminal `и`

`и$` → removed (unconditional, single). All refs identical.

### 4.8 Step 3 — DERIVATIONAL → `ость`

Gate regex (Node + Python identical):
`[^аеиоуюяіїє][аеиоуюяіїє]+[^аеиоуюяіїє]+[аеиоуюяіїє].*(?<=о)сть?$`

Read as: RV must contain, in order, C V(+) C(+) V … and end in `ост` or `ость`
(the final `ь` optional, the `с` preceded by `о`). If the shape matches, strip
`ость$`.

> **DIVERGENCE D3 — derivational strip.** The gate matches `ост` *or* `ость`
> (optional ь), but the replacement `ость$` only removes the form *with* ь. So a
> word ending `…ост` passes the gate yet nothing is removed (no-op). the second JS port
> "fixed" this with replacement `ость?$` (also strips `ост`). Canonical follows
> Node+Python (replacement `ость$`, exact): the bare-`ост` case is a no-op here
> and is mopped up later only if some other rule applies. Recorded where it
> matters (e.g. `якість` vs `якост`).

steblo realisation of the gate: walk RV requiring at least
`nonvowel, vowel+, nonvowel+, vowel, …, о, с, т[, ь]$`. Implemented as a small
hand-rolled scan, not a regex, to stay out of the hot path.

### 4.9 Step 4 — `ь`, SUPERLATIVE, doubled `н` — **CANONICAL = Snowball-Russian**

All references *nest* these three operations, and they nest them
two incompatible ways (see the evidence below). We reject **both** nestings and
adopt the canonical **Snowball-Russian Step 4**, which the entire lineage descends
from. Snowball Step 4 is a single choice among three **mutually-exclusive**
branches, tried strictly in this order on RV:

```
(1) if RV ends in нн      → remove the last н  (нн → н)             ; done
(2) else if RV ends in a SUPERLATIVE ending ейш / ейше
                          → remove it, then if it now ends нн, undouble нн → н ; done
(3) else if RV ends in ь  → remove the ь
```

Exactly one branch executes per word. Superlative = `ейш` or `ейше` (trailing `е`
optional). "Undouble" = collapse a terminal `нн` to a single `н`.

Verbatim Snowball wording (Snowball, Russian, Step 4):
> "(1) Undouble н, or, (2) if the word ends with a SUPERLATIVE ending, remove it
> and undouble н, or (3) if the word ends ь (soft sign) remove it."

> **DIVERGENCE D4 — Step 4 (the single most important call in the spec).**
> Canonical adopts **Snowball-Russian Step 4** (the three independent, ordered
> branches above), *not* either the reference nesting.
>
> What the reference implementations do instead (both wrong, in opposite directions):
> - PHP / the early Python reference / the later Python reference:
>   `if remove('ь$') { remove('ейше?$'); нн→н }` — superlative/undouble fire **only
>   when a terminal `ь` was just removed**.
> - Node / the second JS port: the same block but guarded `if NOT removed ь` — the inverse.
>
> Snowball corrects both: undoubling `нн` does not depend on `ь` at all.
>
> Observable consequences vs the refs:
> - `погашення`: RV `гашення` → ADJECTIVE strips `я` → `гашенн`. Snowball branch (1)
>   undoubles → **`погашен`**. (Python keeps `погашенн`; Node also gets `погашен`,
>   so here steblo agrees with Node, disagrees with the Python reference self-test fixture.)
> - a word reaching Step 4 ending in `ь` but **not** `нн`/superlative: branch (3)
>   removes `ь` — same as all refs.
> - a word ending `нн` **and** otherwise also `ь`-eligible: branch (1) wins (`нн`
>   first), `ь` is never reached. This ordering is a deliberate Snowball choice.
>
> The Python reference self-test asserting `погашення → погашенн` is therefore expected to
> *fail* against steblo; that fixture is recorded as a `consensus:false` row citing
> D4 with this rationale. This is a known, intentional departure from the lineage
> roots in favour of the algorithm's true ancestor.

---

## 5. Strict mode — consonant alternation

After all suffix stripping, if `Options.Strict`, apply a terminal
consonant-alternation cleanup. The Node port applies a single regex:

```
([гджзкстхцчш]|ст|дж|ждж|ьц|сі|ці|зі|он|ін|ів|ев|ок|шк)$  →  removed
```

The README documents the *intent* via the classic Ukrainian palatalisation
triples:

```
г ↔ з ↔ ж
к ↔ ц ↔ ч
х ↔ с ↔ ш
```

with examples: `Чепинога → чепиног` (default) → `чепино` (strict);
`Чепинозі → чепино` (strict); and the deliberately-shown over-stripping caveats
`нога → но`, `нозі → но`, `ніжка → ніж` (strict).

> **DIVERGENCE D5 — strict mode is Node-only.** The Python references have no
> strict mode. Canonical adopts the Node behaviour verbatim (the regex above) for
> `Options.Strict`, since it is the only reference that implements it and the
> README pins concrete expected outputs. steblo realises it as a longest-first
> terminal-cluster strip with the same member set. Strict mode is **opt-in** and
> its over-stripping (`нога → но`) is documented as expected, not a bug.

---

## 6. No-vowel / empty-RV short circuit

- If the (preprocessed) word contains **no vowel**, return it unchanged.
- If RV is empty after the split (word is a single leading-vowel cluster with
  nothing after the first vowel), return the word unchanged.

All references agree.

---

## 7. Worked examples (to become corpus seed rows)

| input | prefix | RV in | fires | stem (default) | strict |
|---|---|---|---|---|---|
| випробування | ви | пробування | NOUN `я`→… (`ння`) | випробуван | випробуван |
| Чепинога | Че | пинога | NOUN `а` | чепиног | чепино |
| Чепинозі | Че | пинозі | NOUN `і` | чепиноз | чепино |
| погашення | по | гашення | ADJ `я`, Step4 нн→н | погашен | погашен |
| зберігайте | зб(е)… | …/REFLEXIVE/… | — | зберігайт | зберігайт |
| тямущий | тя | мущий | ADJECTIVE `ий` | тямущ | тямущ |
| рефлексивного | ре | флексивного | ADJECTIVE `ого` | рефлексивн | рефлексивн |
| нога (strict caveat) | но | га | NOUN `а` | ног | но |

(These mirror the reference behaviour. They are the manual seed; exact post-strip
values are re-derived programmatically in the corpus — any mismatch between this
table and the generated corpus is a spec bug to resolve, not to paper over.)

---

## 8. Divergence ledger (summary)

| ID | Topic | Canonical choice | Refs agreeing | Refs differing |
|---|---|---|---|---|
| D1 | perfective gerund в/вши/вшись after а/я | corrected (Node) form | Node | both Python references, the second JS port (latent bug) |
| D2 | adjective suffix set | Python-reference set | both Python references | Node, the second JS port |
| D3 | derivational strip of bare `ост` | no-op (`ость$` exact) | Node, both Python references | the second JS port |
| D4 | Step 4 (ь / ейше / нн) | **Snowball-Russian** (3 ordered independent branches) | none (all 4 refs nest, two ways) | Node + second JS port (one nesting), both Python references+PHP (other) |
| D5 | strict consonant alternation | Node behaviour | Node | both Python references (absent) |

Every `consensus:false` corpus row must cite one of these IDs (or open a new one).

---

## 9a. Non-idempotence (a documented non-property)

`Stem(Stem(x))` is **not** guaranteed to equal `Stem(x)`. This algorithm strips
at most one suffix per phase in a single pass; feeding a stripped form back in
can expose a new strippable ending. This is inherent to the upstream lineage, not
a steblo bug — all references behave identically:

| x | Stem(x) | Stem(Stem(x)) |
|---|---|---|
| прочитавши | прочита | прочит |
| ініціатив | — (already) | ініціат |
| історіє | — | істор |

(Verified: the Node and Python references both produce `прочита→прочит`,
`ініціатив→ініціат`, `історіє→істор`.)

Therefore steblo does **not** ship an idempotence test or fuzz target. The real,
tested invariant instead is **rune-length monotonicity**: the output is never
longer (in runes) than the normalised input. Stemming is a truncation (plus the
`нн→н` undouble and optional normalisation rewrites, all non-lengthening).

## 9. What this stemmer is not

Rule-based truncation, not lemmatisation. It will over- and under-stem (the
strict-mode `нога → но` is shown by the author himself). For true morphology use
VESUM / pymorphy2-uk / Stanza. See README §Caveats.
