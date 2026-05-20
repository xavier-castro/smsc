package config

import (
	"strings"
	"testing"
)

func TestSetYAMLStringPreservesExistingKeys(t *testing.T) {
	input := "nodeLinker: node-modules\n"
	got, err := SetYAMLString(input, "npmMinimalAgeGate", "8d")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "nodeLinker: node-modules") {
		t.Fatalf("expected existing key to remain: %q", got)
	}
	if !strings.Contains(got, "npmMinimalAgeGate: 8d") {
		t.Fatalf("expected npmMinimalAgeGate: %q", got)
	}
	raw, ok, err := ReadYAMLString(got, "npmMinimalAgeGate")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || raw != "8d" {
		t.Fatalf("got raw=%q ok=%v", raw, ok)
	}
}

func TestRemoveYAMLTopKeyPreservesExistingKeys(t *testing.T) {
	input := "# keep me\nnodeLinker: node-modules\nnpmMinimalAgeGate: 8d\n"
	got, err := RemoveYAMLTopKey(input, "npmMinimalAgeGate")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "npmMinimalAgeGate") {
		t.Fatalf("expected npmMinimalAgeGate to be removed: %q", got)
	}
	for _, want := range []string{"# keep me", "nodeLinker: node-modules"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q to be preserved: %q", want, got)
		}
	}
	if noOp, err := RemoveYAMLTopKey(got, "npmMinimalAgeGate"); err != nil || noOp != got {
		t.Fatalf("expected no-op removal to preserve content exactly: got=%q err=%v", noOp, err)
	}
}
