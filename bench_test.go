package steblo

import (
	"bufio"
	"os"
	"testing"
)

func loadBenchWords(b *testing.B) []string {
	b.Helper()
	f, err := os.Open("bench/words.txt")
	if err != nil {
		b.Fatalf("open bench words: %v", err)
	}
	defer f.Close()
	var words []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if w := sc.Text(); w != "" {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		b.Fatal("no bench words")
	}
	return words
}

// BenchmarkStemWorkload stems the full 10k-word workload per iteration and
// reports words/sec via a custom metric.
func BenchmarkStemWorkload(b *testing.B) {
	words := loadBenchWords(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, w := range words {
			_ = Stem(w)
		}
	}
	b.StopTimer()
	wordsPerOp := float64(len(words))
	b.ReportMetric(wordsPerOp*float64(b.N)/b.Elapsed().Seconds(), "words/sec")
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/(wordsPerOp*float64(b.N)), "ns/word")
}

// BenchmarkStemSingle isolates per-call cost on a typical inflected word.
func BenchmarkStemSingle(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Stem("випробування")
	}
}

// BenchmarkStemRunesNoAlloc checks the allocation-conscious path on a word that
// requires no normalisation copy (already lowercase, no apostrophe).
func BenchmarkStemRunesNoAlloc(b *testing.B) {
	w := []rune("випробування")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = StemRunes(w)
	}
}

// BenchmarkStemStrict measures the strict-mode path.
func BenchmarkStemStrict(b *testing.B) {
	opts := DefaultOptions()
	opts.Strict = true
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = StemWith("Чепинозі", opts)
	}
}
