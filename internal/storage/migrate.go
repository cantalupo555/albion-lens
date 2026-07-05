package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// migrationMarker is a sentinel file created in the legacy root after the
// first legacy→region migration completes. Its presence makes subsequent
// migrations (for other regions) no-ops, so the original legacy snapshot is
// not re-copied into every newly detected region (which would seed a Europe
// player's history into Americas, etc.).
const migrationMarker = ".region-migration-done"

// MigrateLegacyData copies all *.json files from the legacy/shared root into
// the given region's subdirectory, exactly once.
//
// Behavior:
//
//   - region "" is a no-op (legacy → legacy has nothing to migrate).
//   - The migration is claimed atomically via an exclusive marker create
//     (O_CREATE|O_EXCL). The first caller to create the marker performs the
//     copy; every other caller — concurrent or in a future session — sees the
//     marker exists and returns immediately. This guarantees the original
//     legacy snapshot is attributed to exactly one region and never re-seeds
//     others, even under concurrent invocation.
//   - Every *.json file in the root is copied into the region dir, skipping any
//     that already exist there (so a partial earlier copy or a user-supplied
//     seed is never overwritten).
//
// The copy (rather than rename) keeps the legacy root intact as a safety net.
// Region subdirectories themselves under the root are never treated as data
// files and are skipped by the *.json glob.
func MigrateLegacyData(region string) error {
	if region == "" {
		return nil
	}

	root, err := DataDirFor("")
	if err != nil {
		return fmt.Errorf("resolve legacy root: %w", err)
	}
	regionDir, err := DataDirFor(region)
	if err != nil {
		return fmt.Errorf("resolve region dir: %w", err)
	}

	// Atomically claim the migration. O_CREATE|O_EXCL fails if the marker
	// already exists, which serializes concurrent callers: only the winner
	// proceeds to copy, everyone else returns. Marker-first prioritizes the
	// "never re-seed other regions" guarantee over retry-on-crash mid-copy
	// (a crash after claiming leaves the region empty; the legacy root remains
	// as the fallback).
	markerPath := filepath.Join(root, migrationMarker)
	fh, err := os.OpenFile(markerPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil // already migrated
		}
		return fmt.Errorf("claim migration marker: %w", err)
	}
	// Record which region won the claim (diagnostic; not read elsewhere).
	_, _ = fh.WriteString(region + "\n")
	_ = fh.Close()

	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read legacy root: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isJSONFile(name) {
			continue
		}
		src := filepath.Join(root, name)
		dst := filepath.Join(regionDir, name)
		// Skip if the region dir already has this file (partial earlier copy
		// or user-supplied seed). Never overwrite existing region data.
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
	}

	return nil
}

// isJSONFile reports whether name ends in ".json" (case-insensitive). It does
// not match the marker file (which has no .json suffix) or directories.
func isJSONFile(name string) bool {
	if len(name) < 6 {
		return false
	}
	ext := name[len(name)-5:]
	return ext == ".json" || ext == ".JSON"
}

// copyFile duplicates src onto dst, preserving nothing beyond contents. It is
// the unit of legacy migration: simple, explicit, and easy to attribute in
// errors. Permissions use 0o644 to match storage.Save's output.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// MigrationDone reports whether the legacy→region migration has already run.
// Exposed for diagnostics and tests; not needed by the normal runtime path.
func MigrationDone() (bool, error) {
	root, err := DataDirFor("")
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(root, migrationMarker))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
