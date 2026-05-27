package bleveuk

import (
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/mapping"
)

func analysisStream(terms ...string) analysis.TokenStream {
	ts := make(analysis.TokenStream, len(terms))
	pos := 0
	for i, term := range terms {
		pos += len(term)
		ts[i] = &analysis.Token{Term: []byte(term), Position: i + 1, Start: pos - len(term), End: pos}
	}
	return ts
}

func TestStemmerFilter(t *testing.T) {
	f := &StemmerFilter{}
	in := analysisStream("випробування", "Чепинога", "погашення")
	out := f.Filter(in)
	want := []string{"випробуван", "чепиног", "погашен"}
	for i, tok := range out {
		if got := string(tok.Term); got != want[i] {
			t.Errorf("token %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestAnalyzerRegistered(t *testing.T) {
	im := mapping.NewIndexMapping()
	a := im.AnalyzerNamed(AnalyzerName)
	if a == nil {
		t.Fatalf("analyzer %q not registered", AnalyzerName)
	}
	// "для" is a stopword and must be dropped; the rest are stemmed.
	tokens := a.Analyze([]byte("випробування для держави"))
	var terms []string
	for _, tk := range tokens {
		terms = append(terms, string(tk.Term))
	}
	if contains(terms, "для") {
		t.Errorf("stopword 'для' survived: %v", terms)
	}
	if !contains(terms, "випробуван") {
		t.Errorf("expected stem 'випробуван' in %v", terms)
	}
}

func TestIndexAndSearch(t *testing.T) {
	im := mapping.NewIndexMapping()
	fm := bleve.NewTextFieldMapping()
	fm.Analyzer = AnalyzerName
	dm := bleve.NewDocumentMapping()
	dm.AddFieldMappingsAt("text", fm)
	im.DefaultMapping = dm

	idx, err := bleve.NewMemOnly(im)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}
	defer idx.Close()

	docs := map[string]string{
		"d1": "державні випробування нового обладнання",
		"d2": "погашення кредиту достроково",
		"d3": "рецепт смачного борщу",
	}
	for id, text := range docs {
		if err := idx.Index(id, map[string]string{"text": text}); err != nil {
			t.Fatalf("index %s: %v", id, err)
		}
	}

	// Query "випробувати" should hit d1 via the shared stem "випробува"/"випробуван".
	cases := []struct {
		query   string
		wantDoc string
	}{
		{"випробування", "d1"},
		{"державних", "d1"},
		{"погашень", "d2"},
	}
	for _, c := range cases {
		q := bleve.NewMatchQuery(c.query)
		q.SetField("text")
		res, err := idx.Search(bleve.NewSearchRequest(q))
		if err != nil {
			t.Fatalf("search %q: %v", c.query, err)
		}
		if res.Total == 0 {
			t.Errorf("query %q returned no hits", c.query)
			continue
		}
		if res.Hits[0].ID != c.wantDoc {
			t.Errorf("query %q top hit = %s, want %s", c.query, res.Hits[0].ID, c.wantDoc)
		}
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
