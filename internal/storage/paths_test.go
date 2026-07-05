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

// TestDataDirForEmptyEqualsDataDir confirms the Unknown-region fallback path
// is exactly the legacy root, so callers can pass "" unconditionally.
func TestDataDirForEmptyEqualsDataDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	legacy, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	empty, err := DataDirFor("")
	if err != nil {
		t.Fatalf("DataDirFor(\"\", err): %v", err)
	}
	if legacy != empty {
		t.Errorf("DataDir() = %q != DataDirFor(\"\") = %q", legacy, empty)
	}
}

func TestDataFileForEmptyEqualsDataFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	legacy, err := DataFile("stats.json")
	if err != nil {
		t.Fatalf("DataFile: %v", err)
	}
	empty, err := DataFileFor("", "stats.json")
	if err != nil {
		t.Fatalf("DataFileFor: %v", err)
	}
	if legacy != empty {
		t.Errorf("DataFile() = %q != DataFileFor(\"\") = %q", legacy, empty)
	}
}

// TestDataDirForRegionNestedUnderRoot confirms per-region dirs live under the
// app root and are created (including missing parents).
func TestDataDirForRegionNestedUnderRoot(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)

	got, err := DataDirFor("UserData-AMERICA")
	if err != nil {
		t.Fatalf("DataDirFor: %v", err)
	}

	want := filepath.Join(base, appName, "UserData-AMERICA")
	if got != want {
		t.Errorf("DataDirFor = %q, want %q", got, want)
	}
	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Errorf("region dir not created at %q (err=%v)", want, err)
	}
}

func TestDataFileForRegionJoinsName(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)

	got, err := DataFileFor("UserData-EUROPE", "session-stats.json")
	if err != nil {
		t.Fatalf("DataFileFor: %v", err)
	}

	want := filepath.Join(base, appName, "UserData-EUROPE", "session-stats.json")
	if got != want {
		t.Errorf("DataFileFor = %q, want %q", got, want)
	}
}

// TestDataDirForIsolation verifies two regions resolve to disjoint paths
// (no cross-region bleed), the core guarantee of region segregation.
func TestDataDirForIsolation(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	am, err := DataDirFor("UserData-AMERICA")
	if err != nil {
		t.Fatalf("AMERICA: %v", err)
	}
	eu, err := DataDirFor("UserData-EUROPE")
	if err != nil {
		t.Fatalf("EUROPE: %v", err)
	}
	legacy, err := DataDirFor("")
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if am == eu || am == legacy || eu == legacy {
		t.Errorf("region paths collide: am=%q eu=%q legacy=%q", am, eu, legacy)
	}
}
