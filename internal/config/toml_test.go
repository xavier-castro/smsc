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

func TestRemoveBunMinimumReleaseAgePreservesInstallSection(t *testing.T) {
	input := "telemetry = false\n\n[install]\nregistry = \"https://registry.npmjs.org\"\nminimumReleaseAge = 691200\n"
	got, err := RemoveBunMinimumReleaseAge(input)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "minimumReleaseAge") {
		t.Fatalf("expected minimumReleaseAge to be removed: %q", got)
	}
	for _, want := range []string{"telemetry = false", "[install]", "registry = \"https://registry.npmjs.org\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q to be preserved: %q", want, got)
		}
	}
	if noOp, err := RemoveBunMinimumReleaseAge(got); err != nil || noOp != got {
		t.Fatalf("expected no-op removal to preserve content exactly: got=%q err=%v", noOp, err)
	}
}

func TestRemoveTOMLTopKeyPreservesSections(t *testing.T) {
	input := "# keep me\nexclude-newer = \"8 days\"\n[pip]\nindex-url = \"https://example.test\"\n"
	got, err := RemoveTOMLTopKey(input, "exclude-newer")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "exclude-newer") {
		t.Fatalf("expected exclude-newer to be removed: %q", got)
	}
	for _, want := range []string{"# keep me", "[pip]", "index-url = \"https://example.test\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q to be preserved: %q", want, got)
		}
	}
}
