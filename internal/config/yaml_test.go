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
