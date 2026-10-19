package i18n

import (
	"sort"
	"testing"
)

func TestCatalogsHaveTheSameKeys(t *testing.T) {
	base := Keys(DefaultLocale)
	sort.Strings(base)
	for _, loc := range Supported {
		for _, k := range base {
			if !Has(loc, k) {
				t.Errorf("%s.json misses %s", loc, k)
			}
		}
		if len(Keys(loc)) != len(base) {
			for _, k := range Keys(loc) {
				if !Has(DefaultLocale, k) {
					t.Errorf("%s.json has extra key %s", loc, k)
				}
			}
		}
	}
}

func TestTReplacesParams(t *testing.T) {
	if got := T("en", "issue.SLA_MISSING", map[string]any{"claim_type": "DENTAL"}); got != "No SLA is set for DENTAL." {
		t.Errorf("got %q", got)
	}
	if got := T("en", "no.such.key", nil); got != "" {
		t.Errorf("unknown key = %q", got)
	}
}
