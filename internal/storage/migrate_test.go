package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMigrateLegacyDataNoOpForEmptyRegion(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := MigrateLegacyData(""); err != nil {
		t.Fatalf("MigrateLegacyData(\"\"): %v", err)
	}
	// No marker should be created for the empty region.
	if done, err := MigrationDone(); err != nil || done {
		t.Errorf("MigrationDone = (%v,%v), want (false,nil)", done, err)
	}
}

func TestMigrateLegacyDataCopiesJSONFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	// Seed legacy data directly in the root.
	legacyRoot := filepath.Join(root, appName)
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "dungeon-runs.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "session-stats.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A non-json file must be ignored.
	if err := os.WriteFile(filepath.Join(legacyRoot, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyData("UserData-AMERICA"); err != nil {
		t.Fatalf("MigrateLegacyData: %v", err)
	}

	regionDir := filepath.Join(legacyRoot, "UserData-AMERICA")
	for _, name := range []string{"dungeon-runs.json", "session-stats.json"} {
		if _, err := os.Stat(filepath.Join(regionDir, name)); err != nil {
			t.Errorf("expected %s migrated into region dir: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(regionDir, "notes.txt")); err == nil {
		t.Error("non-.json file was migrated")
	}

	done, err := MigrationDone()
	if err != nil || !done {
		t.Errorf("MigrationDone = (%v,%v), want (true,nil)", done, err)
	}
}

func TestMigrateLegacyDataIdempotentAcrossRegions(t *testing.T) {
	// The core guarantee: once migrated, legacy data must NOT seed a second
	// region. Otherwise a Europe player's history would leak into Americas.
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	legacyRoot := filepath.Join(root, appName)
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "session-stats.json"),
		[]byte(`{"fame":999}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// First migration: Europe gets the legacy snapshot.
	if err := MigrateLegacyData("UserData-EUROPE"); err != nil {
		t.Fatalf("migrate Europe: %v", err)
	}
	// Mutate the Europe copy so we can tell a re-copy apart.
	if err := os.WriteFile(
		filepath.Join(legacyRoot, "UserData-EUROPE", "session-stats.json"),
		[]byte(`{"fame":111}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second region: Americas must start EMPTY (marker present).
	if err := MigrateLegacyData("UserData-AMERICA"); err != nil {
		t.Fatalf("migrate Americas: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyRoot, "UserData-AMERICA", "session-stats.json")); err == nil {
		t.Error("Americas was seeded from legacy; marker failed to prevent re-migration")
	}
}

func TestMigrateLegacyDataSkipsExistingRegionFiles(t *testing.T) {
	// A region that already has a file must not have it overwritten by the
	// legacy copy (user-supplied or partial seed wins).
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	legacyRoot := filepath.Join(root, appName)
	regionDir := filepath.Join(legacyRoot, "UserData-ASIA")
	if err := os.MkdirAll(regionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regionDir, "session-stats.json"),
		[]byte(`{"fame":777}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "session-stats.json"),
		[]byte(`{"fame":999}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyData("UserData-ASIA"); err != nil {
		t.Fatalf("MigrateLegacyData: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(regionDir, "session-stats.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"fame":777}` {
		t.Errorf("region file overwritten by legacy copy: got %s", got)
	}
}

func TestMigrateLegacyDataIgnoresRegionSubdirs(t *testing.T) {
	// A region subdirectory under the root is not a .json file and must not be
	// copied (would create nested UserData-* copies).
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	legacyRoot := filepath.Join(root, appName)
	if err := os.MkdirAll(filepath.Join(legacyRoot, "UserData-EUROPE"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "dungeon-runs.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyData("UserData-AMERICA"); err != nil {
		t.Fatalf("MigrateLegacyData: %v", err)
	}
	// Only dungeon-runs.json should be copied, not the Europe subdir.
	entries, err := os.ReadDir(filepath.Join(legacyRoot, "UserData-AMERICA"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			t.Errorf("subdirectory %q was copied into region dir", e.Name())
		}
	}
}

// TestMigrateLegacyDataConcurrentRegions verifies that concurrent migration
// calls for different regions are safe (no panic, no half-written files). The
// marker guarantees only the first region receives the legacy data; the others
// must start empty. Run under -race.
func TestMigrateLegacyDataConcurrentRegions(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	legacyRoot := filepath.Join(root, appName)
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "session-stats.json"),
		[]byte(`{"fame":42}`), 0o644); err != nil {
		t.Fatal(err)
	}

	regions := []string{"UserData-AMERICA", "UserData-EUROPE", "UserData-ASIA"}
	var wg sync.WaitGroup
	errs := make(chan error, len(regions))
	for _, r := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			errs <- MigrateLegacyData(region)
		}(r)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent MigrateLegacyData returned error: %v", err)
		}
	}

	// Migration is complete after the race; re-running on the losing regions
	// must be a clean no-op (marker present).
	copied := 0
	for _, r := range regions {
		fp := filepath.Join(legacyRoot, r, "session-stats.json")
		if _, err := os.Stat(fp); err == nil {
			copied++
		}
	}
	// Exactly one region wins the legacy data; the others start empty.
	if copied != 1 {
		t.Errorf("expected exactly 1 region to receive legacy data, got %d", copied)
	}
	// The marker must exist, so future migrations never re-seed.
	done, err := MigrationDone()
	if err != nil || !done {
		t.Errorf("MigrationDone after concurrent run = (%v, %v), want (true, nil)", done, err)
	}
}
