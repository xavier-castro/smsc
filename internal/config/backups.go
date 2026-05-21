package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Backup describes one SMSC backup manifest directory.
type Backup struct {
	Timestamp    string          `json:"timestamp"`
	Path         string          `json:"path"`
	ManifestPath string          `json:"manifestPath"`
	Changes      []AppliedChange `json:"changes"`
}

// RestoredChange describes one path restored from a backup manifest.
type RestoredChange struct {
	Path       string `json:"path"`
	BackupPath string `json:"backupPath,omitempty"`
	Restored   bool   `json:"restored"`
	Removed    bool   `json:"removed"`
}

func BackupRoot(configHome string) string {
	return filepath.Join(configHome, "smsc", "backups")
}

func ListBackups(configHome string) ([]Backup, error) {
	root := BackupRoot(configHome)
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	backups := make([]Backup, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		backup, err := loadBackup(filepath.Join(root, entry.Name()))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp > backups[j].Timestamp
	})
	return backups, nil
}

func ResolveBackup(configHome, spec string) (Backup, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return Backup{}, errors.New("backup timestamp is required")
	}
	if spec == "latest" {
		backups, err := ListBackups(configHome)
		if err != nil {
			return Backup{}, err
		}
		if len(backups) == 0 {
			return Backup{}, errors.New("no SMSC backups found")
		}
		return backups[0], nil
	}
	if filepath.Base(spec) != spec || spec == "." || spec == ".." {
		return Backup{}, fmt.Errorf("invalid backup timestamp %q", spec)
	}
	return loadBackup(filepath.Join(BackupRoot(configHome), spec))
}

func RestoreBackup(backup Backup) ([]RestoredChange, error) {
	restored := make([]RestoredChange, 0, len(backup.Changes))
	for _, change := range backup.Changes {
		if !change.Changed {
			continue
		}
		if strings.TrimSpace(change.Path) == "" {
			return restored, errors.New("backup manifest contains an empty path")
		}

		if change.BackupPath == "" {
			if err := os.Remove(change.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return restored, err
			}
			restored = append(restored, RestoredChange{Path: change.Path, Removed: true})
			continue
		}

		backupPath := change.BackupPath
		if !filepath.IsAbs(backupPath) {
			backupPath = filepath.Join(backup.Path, backupPath)
		}
		data, err := os.ReadFile(backupPath)
		if err != nil {
			return restored, err
		}
		if err := os.MkdirAll(filepath.Dir(change.Path), 0o755); err != nil {
			return restored, err
		}
		if err := os.WriteFile(change.Path, data, 0o644); err != nil {
			return restored, err
		}
		restored = append(restored, RestoredChange{Path: change.Path, BackupPath: backupPath, Restored: true})
	}
	return restored, nil
}

func loadBackup(dir string) (Backup, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Backup{}, err
	}
	var changes []AppliedChange
	if err := json.Unmarshal(data, &changes); err != nil {
		return Backup{}, err
	}
	return Backup{
		Timestamp:    filepath.Base(dir),
		Path:         dir,
		ManifestPath: manifestPath,
		Changes:      changes,
	}, nil
}
