// Command stemctl stems Ukrainian text from stdin using steblo.
//
//	echo "слова українські красиві" | stemctl      → слов українськ красив
//	stemctl --strict --json < words.txt            → {"слова":"слов", ...}
//	stemctl --bench < bench/words.txt              → throughput report
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/tggo/steblo"
)

func main() {
	var (
		strict      = flag.Bool("strict", false, "apply consonant-alternation strict mode")
		jsonOut     = flag.Bool("json", false, "emit a JSON object mapping word→stem")
		bench       = flag.Bool("bench", false, "stem all input words and report throughput")
		normalizeYo = flag.Bool("normalize-yo", false, "map Russian ё→е and ъ→ї")
	)
	flag.Parse()

	opts := steblo.DefaultOptions()
	opts.Strict = *strict
	opts.NormalizeYo = *normalizeYo

	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stemctl: read stdin:", err)
		os.Exit(1)
	}
	words := strings.Fields(string(in))

	switch {
	case *bench:
		runBench(words, opts)
	case *jsonOut:
		runJSON(words, opts)
	default:
		out := make([]string, len(words))
		for i, w := range words {
			out[i] = steblo.StemWith(w, opts)
		}
		w := bufio.NewWriter(os.Stdout)
		defer w.Flush()
		fmt.Fprintln(w, strings.Join(out, " "))
	}
}

// runJSON prints an ordered JSON object (first-occurrence order, deduplicated).
func runJSON(words []string, opts steblo.Options) {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	seen := make(map[string]bool, len(words))
	w.WriteByte('{')
	first := true
	for _, word := range words {
		if seen[word] {
			continue
		}
		seen[word] = true
		if !first {
			w.WriteByte(',')
		}
		first = false
		fmt.Fprintf(w, "%s:%s", jsonString(word), jsonString(steblo.StemWith(word, opts)))
	}
	w.WriteByte('}')
	w.WriteByte('\n')
}

// jsonString escapes a string as a JSON string literal (UTF-8 passed through).
func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func runBench(words []string, opts steblo.Options) {
	if len(words) == 0 {
		fmt.Fprintln(os.Stderr, "stemctl: no words on stdin")
		os.Exit(1)
	}
	// Retain results in a sink so escape analysis cannot elide the output
	// allocation — this measures real usage, not a discarded-result micro-loop.
	sink := make([]string, len(words))
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	for i, w := range words {
		sink[i] = steblo.StemWith(w, opts)
	}
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(sink)

	n := float64(len(words))
	allocs := float64(after.Mallocs - before.Mallocs)
	fmt.Printf("%d words, %s total, %.0f ns/word, %.1f allocs/word\n",
		len(words), elapsed.Round(time.Microsecond), float64(elapsed.Nanoseconds())/n, allocs/n)
}
