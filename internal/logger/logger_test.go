package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesConfiguredLogFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beebuzz.log")
	t.Setenv("BEEBUZZ_LOG_FILE", path)

	log := New("development")
	log.Info("billing webhook processed", "event_id", "evt_test", "outcome", "applied")

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, want := range []string{"billing webhook processed", "event_id=evt_test", "outcome=applied"} {
		if !strings.Contains(string(contents), want) {
			t.Fatalf("log file missing %q: %s", want, contents)
		}
	}
}

func TestNewUsesInfoLevelForProductionWithLogFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "beebuzz.json.log")
	t.Setenv("BEEBUZZ_LOG_FILE", path)

	log := New("production")
	log.Debug("not written")
	log.Info("written")

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(contents), "not written") || !strings.Contains(string(contents), "written") {
		t.Fatalf("unexpected production log contents: %s", contents)
	}
}
