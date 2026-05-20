package config

import (
	"strings"
	"testing"
)

func TestSetBunMinimumReleaseAgePreservesInstallSection(t *testing.T) {
	input := "telemetry = false\n\n[install]\nregistry = \"https://registry.npmjs.org\"\n"
	got, err := SetBunMinimumReleaseAge(input, 691200)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "telemetry = false") {
		t.Fatalf("expected top-level key: %q", got)
	}
	if !strings.Contains(got, "registry = \"https://registry.npmjs.org\"") {
		t.Fatalf("expected install registry: %q", got)
	}
	if !strings.Contains(got, "minimumReleaseAge = 691200") {
		t.Fatalf("expected minimumReleaseAge: %q", got)
	}
	seconds, _, ok, err := ReadBunMinimumReleaseAge(got)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || seconds != 691200 {
		t.Fatalf("got seconds=%d ok=%v", seconds, ok)
	}
}

func TestSetTOMLTopStringPreservesSections(t *testing.T) {
	input := "[pip]\nindex-url = \"https://example.test\"\n"
	got, err := SetTOMLTopString(input, "exclude-newer", "8 days")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "exclude-newer = \"8 days\"\n[pip]") {
		t.Fatalf("unexpected output: %q", got)
	}
}
