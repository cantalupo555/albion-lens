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

// DataDir resolves the application data directory following the XDG Base
// Directory Specification: $XDG_DATA_HOME/albion-lens, falling back to
// ~/.local/share/albion-lens when XDG_DATA_HOME is unset. The directory (and
// any missing parents) are created with mode 0755.
func DataDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// DataFile joins the data directory with the given file name. It is a
// convenience over DataDir for callers that just need a single path.
func DataFile(name string) (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
