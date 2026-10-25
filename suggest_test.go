package kong_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/donvargax/kong"
)

func TestSuggest(t *testing.T) {
	commands := []string{"crap", "dry", "mutation", "scrap", "serve"}
	for _, test := range []struct {
		typed string
		want  []string
	}{
		// Kong's rule: a prefix, or within two edits, in the candidates' order.
		{"crapp", []string{"crap", "scrap"}},
		{"s", []string{"scrap", "serve"}},
		{"dr", []string{"dry"}},
		{"serv", []string{"serve"}},
		{"lint", nil},
		{"scrapers", nil},
		// The stem, when Kong's rule finds none: all but the last letter
		// begins exactly one candidate.
		{"mutate", []string{"mutation"}},
		{"mutax", []string{"mutation"}},
		// Under five letters, the stem is not tried.
		{"mutx", nil},
	} {
		t.Run(test.typed, func(t *testing.T) {
			assert.Equal(t, test.want, kong.Suggest(test.typed, commands))
		})
	}
}

func TestSuggestStemBeginningSeveral(t *testing.T) {
	assert.Equal(t, nil, kong.Suggest("mutate", []string{"mutation", "mutations"}))
}

func TestSuggestStemAfterKongsRuleOnly(t *testing.T) {
	// Kong's rule finds "xbcdef", one edit away, so the stem, which begins
	// only "abcdezzz", is never tried.
	assert.Equal(t, []string{"xbcdef"}, kong.Suggest("abcdef", []string{"abcdezzz", "xbcdef"}))
}
