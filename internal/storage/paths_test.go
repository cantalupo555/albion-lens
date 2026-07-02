package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirRespectsXDGDataHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}

	want := filepath.Join(dir, appName)
	if got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Errorf("DataDir did not create dir %q (err=%v)", want, err)
	}
}

func TestDataDirFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}

	want := filepath.Join(home, ".local", "share", appName)
	if got != want {
		t.Errorf("DataDir = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("fallback dir not created: %v", err)
	}
}

func TestDataFileJoinsName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	got, err := DataFile("dungeon-runs.json")
	if err != nil {
		t.Fatalf("DataFile: %v", err)
	}

	want := filepath.Join(dir, appName, "dungeon-runs.json")
	if got != want {
		t.Errorf("DataFile = %q, want %q", got, want)
	}
}
