package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFileCanRunRepeatedly(t *testing.T) {
	tmp := t.TempDir()
	sourcePath := filepath.Join(tmp, "data.json")
	if err := os.WriteFile(sourcePath, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	first, err := BackupFile(sourcePath)
	if err != nil {
		t.Fatalf("first backup: %v", err)
	}
	second, err := BackupFile(sourcePath)
	if err != nil {
		t.Fatalf("second backup: %v", err)
	}
	if first == second {
		t.Fatalf("backup names must be unique: %s", first)
	}
}
