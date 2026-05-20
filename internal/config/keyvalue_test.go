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

func TestRemoveKeyValuePreservesUnrelatedLines(t *testing.T) {
	input := "# keep me\nregistry=https://example.test\nmin-release-age=8\nbefore=2026-05-12T00:00:00Z\n"
	got := RemoveKeyValue(input, []string{"min-release-age"})
	if strings.Contains(got, "min-release-age") {
		t.Fatalf("expected min-release-age to be removed: %q", got)
	}
	for _, want := range []string{"# keep me", "registry=https://example.test", "before=2026-05-12T00:00:00Z"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q to be preserved: %q", want, got)
		}
	}
	if noOp := RemoveKeyValue(got, []string{"min-release-age"}); noOp != got {
		t.Fatalf("expected no-op removal to preserve content exactly:\nwant %q\ngot  %q", got, noOp)
	}
}

func TestRemoveKeyValueSupportsAliases(t *testing.T) {
	input := "minimumReleaseAge=11520\nminimum-release-age=11520\nregistry=https://example.test\n"
	got := RemoveKeyValue(input, []string{"minimum-release-age", "minimumReleaseAge"})
	if strings.Contains(got, "minimumReleaseAge") || strings.Contains(got, "minimum-release-age") {
		t.Fatalf("expected aliases to be removed: %q", got)
	}
	if !strings.Contains(got, "registry=https://example.test") {
		t.Fatalf("expected unrelated setting to remain: %q", got)
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
