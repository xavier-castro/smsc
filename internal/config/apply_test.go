package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyChangesCreatesBackup(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(root, ".config")
	target := filepath.Join(root, ".npmrc")
	if err := os.WriteFile(target, []byte("registry=https://example.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyChanges([]Change{{
		Path:   target,
		Before: "registry=https://example.test\n",
		After:  "registry=https://example.test\nmin-release-age=8\n",
	}}, configHome, time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || !applied[0].Changed || applied[0].BackupPath == "" {
		t.Fatalf("unexpected applied result: %#v", applied)
	}
	if _, err := os.Stat(applied[0].BackupPath); err != nil {
		t.Fatalf("expected backup to exist: %v", err)
	}
}
