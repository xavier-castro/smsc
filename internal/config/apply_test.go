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

func TestListAndRestoreBackup(t *testing.T) {
	root := t.TempDir()
	configHome := filepath.Join(root, ".config")
	target := filepath.Join(root, ".npmrc")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ApplyChanges([]Change{{Path: target, Before: "", After: "min-release-age=8\n"}}, configHome, time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	backups, err := ListBackups(configHome)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 || backups[0].Timestamp != "20260520T120000Z" {
		t.Fatalf("unexpected backups: %#v", backups)
	}
	if backups[0].Changes[0].BackupPath == "" {
		t.Fatalf("expected empty existing file to have a backup path: %#v", backups[0].Changes[0])
	}
	restored, err := RestoreBackup(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 || !restored[0].Restored {
		t.Fatalf("unexpected restore result: %#v", restored)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty file to be restored, got %q", string(got))
	}
}
