package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"--version"}, &out, &err)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, err.String())
	}
	if out.String() == "" {
		t.Fatal("expected version output")
	}
}

func TestJSONDryRun(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run([]string{"--json", "--dry-run"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if decoded["days"].(float64) != 8 {
		t.Fatalf("unexpected days: %#v", decoded["days"])
	}
}
