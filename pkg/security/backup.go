package security

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func BackupFile(path string) (string, error) {
	startedAt := time.Now()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("backup skipped because source does not exist", "path", path)
			return "", nil
		}
		return "", fmt.Errorf("stat source: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("source is a directory: %s", path)
	}

	source, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer source.Close()

	backupDir := filepath.Join(filepath.Dir(path), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}

	backupPath := filepath.Join(backupDir, filepath.Base(path)+"."+time.Now().UTC().Format("20060102T150405.000000000Z")+".bak")
	target, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create backup: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return "", fmt.Errorf("copy backup: %w", err)
	}
	if err := target.Sync(); err != nil {
		return "", fmt.Errorf("sync backup: %w", err)
	}

	slog.Info("backup created", "source", path, "backup", backupPath, "bytes", info.Size(), "duration", time.Since(startedAt))
	return backupPath, nil
}
