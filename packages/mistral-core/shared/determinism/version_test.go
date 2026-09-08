package determinism

import "testing"

func TestCanonicalPreservesLegacyV1Contract(t *testing.T) {
	for _, input := range []Version{"", RulesV1} {
		resolved, err := Canonical(input)
		if err != nil {
			t.Fatal(err)
		}
		if resolved != RulesV1 {
			t.Fatalf("canonical version = %q, want %q", resolved, RulesV1)
		}
	}
	if _, err := Canonical("mistral.rules.future"); err == nil {
		t.Fatal("unsupported ruleset version was accepted")
	}
}
