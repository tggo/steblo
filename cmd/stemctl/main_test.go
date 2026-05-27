package main

import (
	"testing"

	"github.com/tggo/steblo"
)

func TestRenderJSON(t *testing.T) {
	opts := steblo.DefaultOptions()
	got := renderJSON([]string{"слова", "слова", "Чепинога"}, opts)
	want := `{"слова":"слов","Чепинога":"чепиног"}` + "\n"
	if got != want {
		t.Errorf("renderJSON =\n  %q\nwant\n  %q", got, want)
	}
}

func TestRenderJSONEscaping(t *testing.T) {
	got := renderJSON([]string{`a"b`}, steblo.DefaultOptions())
	if want := `{"a\"b":"a\"b"}` + "\n"; got != want {
		t.Errorf("renderJSON escaping = %q, want %q", got, want)
	}
}

func TestJSONString(t *testing.T) {
	for in, want := range map[string]string{
		"abc":   `"abc"`,
		"a\"b":  `"a\"b"`,
		"a\\b":  `"a\\b"`,
		"l\tn":  `"l\tn"`,
		"слово": `"слово"`,
	} {
		if got := jsonString(in); got != want {
			t.Errorf("jsonString(%q) = %q, want %q", in, got, want)
		}
	}
}
