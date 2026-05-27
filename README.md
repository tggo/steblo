# steblo

Zero-dependency, rule-based stemmer for the Ukrainian language in pure Go.

- **Zero runtime dependencies** — no cgo, no models, no regex in the hot path.
- **Concurrency-safe** — stateless, no package-level mutable state.
- **Allocation-free** hot path via `StemRunes` (~127 ns/word; ~4.9M words/sec).
- Optional Bleve analyzer in the decoupled [`bleveuk`](./bleveuk) sub-package.

The canonical algorithm and every design decision live in
[`docs/algorithm.md`](./docs/algorithm.md) — that spec, not the code, is the
source of truth.

## Install

```bash
go get github.com/tggo/steblo
go install github.com/tggo/steblo/cmd/stemctl@latest   # CLI
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/tggo/steblo"
)

func main() {
	fmt.Println(steblo.Stem("випробування")) // випробуван

	fmt.Println(steblo.StemWith("Чепинога", steblo.Options{Strict: true})) // чепино
}
```

CLI:

```bash
echo "слова українські красиві" | stemctl     # слов українськ красив
stemctl --strict --json < words.txt           # {"слова":"слов", ...}
stemctl --bench < bench/words.txt             # 10000 words, 2.2ms total, 219 ns/word, 1.0 allocs/word
```

## Options

`StemWith(word, Options{…})`. Defaults shown are those used by `Stem`.

| Option | Default | Effect |
|---|---|---|
| `Strict` | `false` | Apply consonant-alternation cleanup after stripping (e.g. `Чепинозі → чепино`). Over-strips by design. |
| `Lowercase` | `true` | Pre-lowercase via `unicode.ToLower`. |
| `NormalizeApostr` | `true` | Unify and delete apostrophe variants before stemming (`об'єднання → обєднан`). |
| `NormalizeYo` | `false` | Map `ё→е`, `ъ→ї` for mixed-Cyrillic corpora. |

Decomposed (NFD) Cyrillic — combining breve (`й`) and diaeresis (`ї`, `ё`) — is
always recomposed, so input from NFD sources (e.g. macOS filenames) stems
identically to NFC. This is unconditional and never alters NFC text.

API surface: `Stem`, `StemWith`, `StemRunes`, `StemRunesWith`, `Options`,
`DefaultOptions`. `StemRunes*` is the allocation-conscious form; its result may
alias the input — clone before mutating.

## Performance

| | ns/word | allocs/word |
|---|---:|---:|
| `Stem` (string API) | ~204 | 1 |
| `StemRunes` (no normalisation copy) | ~127 | **0** |

Measured on Apple M4 Max, Go 1.25, over `bench/words.txt`. Full methodology and
numbers in [`docs/bench.md`](./docs/bench.md). Reproduce with `make bench`.

## Bleve integration

The [`bleveuk`](./bleveuk) sub-package registers a Bleve token filter
(`uk_stem`), a Ukrainian stopword filter (`stop_uk`), and an analyzer (`uk`)
composed of `unicode → lowercase → stop_uk → uk_stem`. Import it for its
side effects, then reference the analyzer by name in your field mapping:

```go
package main

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/tggo/steblo/bleveuk" // registers the "uk" analyzer
)

func main() {
	// Map a text field to the Ukrainian analyzer.
	fm := bleve.NewTextFieldMapping()
	fm.Analyzer = "uk"
	dm := bleve.NewDocumentMapping()
	dm.AddFieldMappingsAt("text", fm)
	im := bleve.NewIndexMapping()
	im.DefaultMapping = dm

	idx, _ := bleve.NewMemOnly(im)
	idx.Index("d1", map[string]string{"text": "державні випробування обладнання"})
	idx.Index("d2", map[string]string{"text": "погашення кредиту достроково"})

	// "випробувань" and the indexed "випробування" both stem to "випробуван",
	// so the query matches d1 even though the surface forms differ.
	q := bleve.NewMatchQuery("випробувань")
	q.SetField("text")
	res, _ := idx.Search(bleve.NewSearchRequest(q))
	fmt.Println(res.Hits[0].ID) // d1
}
```

`bleveuk` is a **separate Go module** (`github.com/tggo/steblo/bleveuk`) so that
Bleve's dependency tree never touches the core: `go get github.com/tggo/steblo`
pulls in nothing. Install the integration only if you want it:

```bash
go get github.com/tggo/steblo/bleveuk
```

## Caveats

steblo is a **rule-based truncation stemmer, not a lemmatiser**. It deliberately
over- and under-stems. It does no dictionary lookup, no morphological analysis,
and no mixed-script (Ukrainian/Russian) disambiguation — the caller must detect
script. If you need lemmas, POS, or full paradigms, use a morphological analyser.

`Stem` is **not idempotent**: `Stem(Stem(x))` may differ from `Stem(x)`, because
each call strips one suffix per phase. See
[`docs/algorithm.md`](./docs/algorithm.md) §9a.

## Development

```bash
make test      # unit + corpus differential tests
make cover     # coverage (core package > 90%)
make bench     # benchmarks
make fuzz      # fuzz targets
make lint      # go vet + staticcheck + golangci-lint if installed
```

## Not in the public repo

Kept local only (gitignored):

- `CLAUDE.md` — internal instructions, full of external links.
- `scripts/build_corpus/` — the corpus generator references external repos by
  name; the generated corpus ships, the generator doesn't.

## Sources

Algorithm lineage and reference implementations:

- [drupal ukstemmer](https://www.drupal.org/project/ukstemmer)
- [Amice13/ukr_stemmer](https://github.com/Amice13/ukr_stemmer) · [ukrstemmer-node](https://github.com/Amice13/ukrstemmer-node)
- [Desklop/Uk_Stemmer](https://github.com/Desklop/Uk_Stemmer)
- [titarenko/ukrstemmer](https://github.com/titarenko/ukrstemmer)
- corpus seeded from [brown-uk/corpus](https://github.com/brown-uk/corpus)

## License

MIT — see [`LICENSE`](./LICENSE).
