# steblo — benchmark report

Methodology: single machine, single-thread, warm cache, `go test -bench=. -benchmem
-count=6`, median reported. No cross-implementation comparison is drawn here — these
are steblo's absolute numbers. Workload is the committed `bench/words.txt` (10,000
frequency-ranked Ukrainian word forms).

## Environment

- CPU: Apple M4 Max (`arm64`), `goos: darwin`
- Go: 1.25
- Reproduce: `make bench`

## Results

| benchmark | ns/op | B/op | allocs/op | notes |
|---|---:|---:|---:|---|
| `StemWorkload` (10k words) | 2,040,000 | 202,240 | 10,125 | whole list per iteration |
| └ per word | **204 ns/word** | ~20 B | ~1.0 | **4.9M words/sec** |
| `StemSingle` (`Stem` string API) | 200 | 24 | 1 | one inflected word |
| `StemRunes` (no normalisation copy) | **127** | **0** | **0** | allocation-free hot path |
| `StemStrict` | 225 | 56 | 2 | strict mode, with normalisation copy |

## Where the allocations go

- `StemRunes([]rune)` on an already-normalised word is **zero-allocation**: every
  phase only truncates the working region from the right, so the stem is a
  left-anchored sub-slice of the input — no reassembly, no copy. The input
  `[]rune` buffer stays on the stack (it does not escape).
- `Stem(string)` costs exactly **one allocation**: the output string. The
  `[]rune(word)` conversion is stack-allocated by escape analysis; only the final
  `string(stem)` escapes.
- Strict mode and `NormalizeApostr`/`NormalizeYo` that actually fire add one copy
  for the normalised rune buffer.

## Characteristics

- ~204 ns/word end-to-end through the string API on this corpus; ~127 ns through
  the rune API with no normalisation. The difference is dominated by the
  `string`↔`[]rune` boundary, not the stemming logic.
- No regex in the hot path (regex is confined to nothing — normalisation is a
  rune scan). Suffix matching is a longest-first linear scan over small tables;
  it has not warranted a trie at this scale.
- Throughput scales linearly with goroutines: the package has no shared mutable
  state, so N cores ≈ N× throughput (not benchmarked here; the API is documented
  safe for concurrent use).

## Honest notes

- `B/op` for the workload (~20 B/word) is the output strings; callers that can
  consume `[]rune` via `StemRunes` and avoid materialising strings pay zero.
- These are absolute single-implementation numbers; no cross-implementation
  comparison is drawn.
