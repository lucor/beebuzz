package database

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewTightensPermissions verifies that opening the SQLite database
// leaves the directory at 0700 and any existing companion files at 0600,
// even when the directory was previously created with a looser mode.
func TestNewTightensPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes not enforced on Windows")
	}

	dir := t.TempDir()
	// Simulate a directory created by an older release with the looser mode.
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("seed dir mode: %v", err)
	}

	db, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Force a write so SQLite materializes the WAL companion files.
	if _, err := db.Exec("CREATE TABLE perms_probe(x INTEGER)"); err != nil {
		t.Fatalf("create probe table: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("dir mode = %o, want 0700", got)
	}

	for _, suffix := range []string{"", "-shm", "-wal"} {
		path := filepath.Join(dir, dbFileName+suffix)
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("stat %s: %v", path, err)
		}
		if got := fi.Mode().Perm(); got != 0o600 {
			t.Errorf("%s mode = %o, want 0600", filepath.Base(path), got)
		}
	}
}
