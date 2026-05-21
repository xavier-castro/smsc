package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Change struct {
	ManagerID   string `json:"managerId"`
	ManagerName string `json:"managerName"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Before      string `json:"-"`
	After       string `json:"-"`
}

type AppliedChange struct {
	Path       string `json:"path"`
	BackupPath string `json:"backupPath,omitempty"`
	Changed    bool   `json:"changed"`
}

func ReadFile(path string) (content string, exists bool, err error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return string(data), true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	return "", false, err
}

func ApplyChanges(changes []Change, configHome string, now time.Time) ([]AppliedChange, error) {
	merged := MergeChanges(changes)
	if len(merged) == 0 {
		return nil, nil
	}

	backupDir := filepath.Join(BackupRoot(configHome), now.UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, err
	}

	applied := make([]AppliedChange, 0, len(merged))
	for _, change := range merged {
		if change.Before == change.After {
			applied = append(applied, AppliedChange{Path: change.Path, Changed: false})
			continue
		}

		if err := os.MkdirAll(filepath.Dir(change.Path), 0o755); err != nil {
			return applied, err
		}

		backupPath := ""
		if _, err := os.Stat(change.Path); err == nil {
			backupPath = filepath.Join(backupDir, sanitizePath(change.Path))
			if err := os.WriteFile(backupPath, []byte(change.Before), 0o600); err != nil {
				return applied, err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return applied, err
		}

		if err := os.WriteFile(change.Path, []byte(change.After), 0o644); err != nil {
			return applied, err
		}
		applied = append(applied, AppliedChange{Path: change.Path, BackupPath: backupPath, Changed: true})
	}

	if len(applied) > 0 {
		if err := writeManifest(backupDir, applied); err != nil {
			return applied, err
		}
	}
	return applied, nil
}

func MergeChanges(changes []Change) []Change {
	byPath := map[string]Change{}
	order := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.Path == "" || change.Before == change.After {
			continue
		}
		existing, ok := byPath[change.Path]
		if !ok {
			byPath[change.Path] = change
			order = append(order, change.Path)
			continue
		}
		if existing.After == change.After {
			existing.Description += "; " + change.Description
			byPath[change.Path] = existing
			continue
		}
		// Last writer wins only when the starting point matches. This keeps
		// multi-manager plans deterministic while avoiding partial patch merges.
		existing.After = change.After
		existing.Description += "; " + change.Description
		byPath[change.Path] = existing
	}

	merged := make([]Change, 0, len(order))
	for _, path := range order {
		merged = append(merged, byPath[path])
	}
	return merged
}

func sanitizePath(path string) string {
	cleaned := strings.TrimPrefix(filepath.Clean(path), string(os.PathSeparator))
	cleaned = strings.ReplaceAll(cleaned, string(os.PathSeparator), "__")
	if cleaned == "" {
		return "root"
	}
	return cleaned
}

func writeManifest(backupDir string, applied []AppliedChange) error {
	if backupDir == "" {
		return nil
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(applied, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(backupDir, "manifest.json"), data, 0o600)
}
