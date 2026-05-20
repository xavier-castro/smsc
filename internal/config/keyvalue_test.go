package config

import (
	"strings"
	"testing"
)

func TestUpsertKeyValuePreservesUnrelatedLinesAndDisablesConflict(t *testing.T) {
	input := "# keep me\nregistry=https://example.test\nbefore=2026-05-12T00:00:00Z\n"
	got := UpsertKeyValue(input, "min-release-age", "8", nil, []string{"before"})
	if !strings.Contains(got, "# keep me") {
		t.Fatalf("expected comment to be preserved: %q", got)
	}
	if !strings.Contains(got, "registry=https://example.test") {
		t.Fatalf("expected unrelated setting to be preserved: %q", got)
	}
	if !strings.Contains(got, "# smsc disabled conflicting setting: before=2026-05-12T00:00:00Z") {
		t.Fatalf("expected before setting to be disabled: %q", got)
	}
	if !strings.Contains(got, "min-release-age=8") {
		t.Fatalf("expected min-release-age to be added: %q", got)
	}
}

func TestParseAgeDuration(t *testing.T) {
	cases := map[string]int64{
		"8d":       8 * DaySeconds,
		"8 days":   8 * DaySeconds,
		"1 week":   7 * DaySeconds,
		"PT24H":    DaySeconds,
		"P7D":      7 * DaySeconds,
		"48 hours": 2 * DaySeconds,
	}
	for input, want := range cases {
		got, ok := ParseAgeDuration(input)
		if !ok {
			t.Fatalf("expected %q to parse", input)
		}
		if got != want {
			t.Fatalf("%q: got %d want %d", input, got, want)
		}
	}
}
