package steblo

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// corpusRow is one case from corpus/cases.yaml.
type corpusRow struct {
	Word, Stem, StrictStem string
	Consensus              bool
	Divergence             string
}

// loadCorpus parses corpus/cases.yaml. The file is machine-generated with a
// fixed, line-oriented schema, so a tiny purpose-built parser keeps the steblo
// package dependency-free (no YAML library, runtime or test).
func loadCorpus(t *testing.T) []corpusRow {
	t.Helper()
	f, err := os.Open("corpus/cases.yaml")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer f.Close()

	var rows []corpusRow
	var cur *corpusRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if v, ok := scalar(line, "- word:"); ok {
			rows = append(rows, corpusRow{Word: v})
			cur = &rows[len(rows)-1]
			continue
		}
		if cur == nil {
			continue
		}
		if v, ok := scalar(line, "stem:"); ok {
			cur.Stem = v
		} else if v, ok := scalar(line, "strict_stem:"); ok {
			cur.StrictStem = v
		} else if v, ok := scalar(line, "consensus:"); ok {
			cur.Consensus = v == "true"
		} else if v, ok := scalar(line, "divergence:"); ok {
			cur.Divergence = v
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan corpus: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("corpus is empty")
	}
	return rows
}

// scalar extracts a value for `key` from a line, unquoting a single-quoted YAML
// scalar ('' -> '). Returns ok=false if the trimmed line does not start with key.
func scalar(line, key string) (string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, key) {
		return "", false
	}
	v := strings.TrimSpace(strings.TrimPrefix(t, key))
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		v = strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return v, true
}

// TestCorpusConsensus is the load-bearing CI gate: every consensus row (the two
// primary references agree and steblo matched them at generation time) must
// still stem as recorded. A failure means either a stemmer regression or an
// intended behaviour change that requires regenerating the corpus + review.
func TestCorpusConsensus(t *testing.T) {
	strictOpts := DefaultOptions()
	strictOpts.Strict = true

	n := 0
	for _, r := range loadCorpus(t) {
		if !r.Consensus {
			continue
		}
		n++
		if got := Stem(r.Word); got != r.Stem {
			t.Errorf("Stem(%q) = %q, want %q", r.Word, got, r.Stem)
		}
		if got := StemWith(r.Word, strictOpts); got != r.StrictStem {
			t.Errorf("StemWith(%q, strict) = %q, want %q", r.Word, got, r.StrictStem)
		}
	}
	t.Logf("checked %d consensus rows", n)
}

// TestCorpusDivergence is informational: it never fails. It records, per
// divergence class, whether steblo still produces the recorded stem (drift
// detection) so a human can review intentional vs accidental change.
func TestCorpusDivergence(t *testing.T) {
	byClass := map[string]int{}
	drift := 0
	for _, r := range loadCorpus(t) {
		if r.Consensus {
			continue
		}
		byClass[r.Divergence]++
		if got := Stem(r.Word); got != r.Stem {
			drift++
			t.Logf("DRIFT [%s] Stem(%q) = %q, recorded %q", r.Divergence, r.Word, got, r.Stem)
		}
	}
	t.Logf("divergence rows by class: %v", byClass)
	if drift > 0 {
		t.Logf("%d divergence rows drifted from recorded output (informational)", drift)
	}
}

// TestCorpusLengthMonotonic checks the real invariant (idempotence does NOT
// hold for this algorithm — see docs/algorithm.md §9a): the stem is never
// longer, in runes, than the word.
func TestCorpusLengthMonotonic(t *testing.T) {
	for _, r := range loadCorpus(t) {
		if got := len([]rune(Stem(r.Word))); got > len([]rune(r.Word)) {
			t.Errorf("Stem(%q) grew: %d > %d runes", r.Word, got, len([]rune(r.Word)))
		}
	}
}
