// Package bleveuk registers a Bleve token filter and analyzer for Ukrainian,
// backed by the steblo stemmer. Importing this package for its side effects
// (registration) is enough:
//
//	import _ "github.com/tggo/steblo/bleveuk"
//
// then reference the analyzer by name "uk" in a Bleve mapping. The core steblo
// package has no Bleve dependency; this sub-package is decoupled.
package bleveuk

import (
	_ "embed"
	"strings"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/token/stop"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/tggo/steblo"
)

const (
	// StemmerName is the registered token filter name.
	StemmerName = "uk_stem"
	// StopName is the registered Ukrainian stopword token filter name.
	StopName = "stop_uk"
	// StopMapName is the registered Ukrainian stopword token map name.
	StopMapName = "stopwords_uk"
	// AnalyzerName is the registered analyzer name.
	AnalyzerName = "uk"
)

//go:embed data/stopwords_uk.txt
var stopwordsData string

// StemmerFilter applies steblo.Stem to each token's term.
type StemmerFilter struct{}

// Filter stems every token in place and returns the same stream.
func (f *StemmerFilter) Filter(input analysis.TokenStream) analysis.TokenStream {
	for _, tok := range input {
		stemmed := steblo.Stem(string(tok.Term))
		tok.Term = []byte(stemmed)
	}
	return input
}

func stemmerConstructor(_ map[string]interface{}, _ *registry.Cache) (analysis.TokenFilter, error) {
	return &StemmerFilter{}, nil
}

// stopTokenMap builds the Ukrainian stopword token map from the embedded list.
func stopTokenMap(_ map[string]interface{}, _ *registry.Cache) (analysis.TokenMap, error) {
	tm := analysis.NewTokenMap()
	for _, line := range strings.Split(stopwordsData, "\n") {
		w := strings.TrimSpace(line)
		if w == "" || strings.HasPrefix(w, "#") {
			continue
		}
		tm.AddToken(w)
	}
	return tm, nil
}

func stopFilterConstructor(_ map[string]interface{}, cache *registry.Cache) (analysis.TokenFilter, error) {
	tm, err := cache.TokenMapNamed(StopMapName)
	if err != nil {
		return nil, err
	}
	return stop.NewStopTokensFilter(tm), nil
}

// analyzerConstructor wires: unicode tokenizer → to_lower → stop_uk → uk_stem.
func analyzerConstructor(_ map[string]interface{}, cache *registry.Cache) (analysis.Analyzer, error) {
	tokenizer, err := cache.TokenizerNamed(unicode.Name)
	if err != nil {
		return nil, err
	}
	toLower, err := cache.TokenFilterNamed(lowercase.Name)
	if err != nil {
		return nil, err
	}
	stopFilter, err := cache.TokenFilterNamed(StopName)
	if err != nil {
		return nil, err
	}
	stemmer, err := cache.TokenFilterNamed(StemmerName)
	if err != nil {
		return nil, err
	}
	return &analysis.DefaultAnalyzer{
		Tokenizer:    tokenizer,
		TokenFilters: []analysis.TokenFilter{toLower, stopFilter, stemmer},
	}, nil
}

func init() {
	registry.RegisterTokenMap(StopMapName, stopTokenMap)
	registry.RegisterTokenFilter(StopName, stopFilterConstructor)
	registry.RegisterTokenFilter(StemmerName, stemmerConstructor)
	registry.RegisterAnalyzer(AnalyzerName, analyzerConstructor)
}
