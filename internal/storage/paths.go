// Package storage provides atomic JSON persistence for app state, porting the
// reference repository's FileController (atomic tmp+rename writes with crash
// recovery on load) to an XDG-compliant data directory.
package storage

import (
	"os"
	"path/filepath"
)

// appName is the per-application subdirectory under the XDG data dir.
const appName = "albion-lens"

// xdgBase resolves the XDG data root: $XDG_DATA_HOME when set, otherwise
// ~/.local/share. It performs no filesystem access and is shared by the legacy
// and per-region path resolvers.
func xdgBase() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return base, nil
}

// DataDir resolves the legacy/shared application data directory following the
// XDG Base Directory Specification: $XDG_DATA_HOME/albion-lens, falling back to
// ~/.local/share/albion-lens when XDG_DATA_HOME is unset. The directory (and
// any missing parents) are created with mode 0755.
//
// This is the Unknown-region fallback and is identical to DataDirFor("").
func DataDir() (string, error) {
	return DataDirFor("")
}

// DataFile joins the legacy data directory with the given file name. It is a
// convenience over DataDir for callers that just need a single path, and is
// identical to DataFileFor("", name).
func DataFile(name string) (string, error) {
	return DataFileFor("", name)
}

// DataDirFor resolves a per-region application data directory:
//
//	$XDG_DATA_HOME/albion-lens/<region>/   when region != ""
//	$XDG_DATA_HOME/albion-lens/            when region == "" (legacy fallback)
//
// An empty region yields the shared root (identical to DataDir), so callers
// with no active region pass "" without special-casing. The region is the
// on-disk subdirectory name (e.g. "UserData-AMERICA"); it is opaque to storage
// and owned by the caller (serverdetect.ServerLocation.DirName). The directory
// (and any missing parents) are created with mode 0755.
func DataDirFor(region string) (string, error) {
	base, err := xdgBase()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appName)
	if region != "" {
		dir = filepath.Join(dir, region)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// DataFileFor joins a per-region data directory with the given file name. See
// DataDirFor for the region semantics (empty region = legacy/shared root).
func DataFileFor(region, name string) (string, error) {
	dir, err := DataDirFor(region)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
