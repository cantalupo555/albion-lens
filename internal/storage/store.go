package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// locks guards the per-path mutex map.
var (
	locksMu sync.Mutex
	locks   = make(map[string]*sync.Mutex)
)

// lockFor returns a per-path mutex, creating it on first use. Serializing
// writes (and tmp promotion) per path prevents concurrent save/load races on
// the same file without a single global lock.
func lockFor(path string) *sync.Mutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	if l, ok := locks[path]; ok {
		return l
	}
	l := &sync.Mutex{}
	locks[path] = l
	return l
}

// tmpPath returns the temporary-write sibling path for a final path. It lives
// in the same directory so os.Rename is atomic (same filesystem).
func tmpPath(path string) string {
	return path + ".tmp"
}

// Save serializes v as indented JSON and writes it atomically: the data is
// first written to a .tmp sibling and then renamed onto the final path. Rename
// is atomic on the same filesystem, so a crash leaves either the old or the
// new file, never a partially-written one.
func Save(path string, v any) error {
	l := lockFor(path)
	l.Lock()
	defer l.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	tmp := tmpPath(path)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Best-effort cleanup of the orphaned tmp file.
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Load reads and decodes the JSON file at path into a new T.
//
// Crash recovery: if a .tmp sibling exists (left over from a Save interrupted
// before the rename), it is tried first — a valid .tmp is promoted to the
// final path, an invalid one is discarded — and then the final path is read.
//
// Return contract (callers must respect both cases):
//   - missing file (and no recoverable .tmp): returns (zero, nil) — the
//     expected first-run case; treat as empty state without warning.
//   - existing file that fails to read or decode: returns (zero, err) — log a
//     warning, since this indicates corruption rather than a first run.
func Load[T any](path string) (T, error) {
	var zero T

	l := lockFor(path)
	l.Lock()
	defer l.Unlock()

	tmp := tmpPath(path)

	// Crash recovery: a leftover .tmp from an interrupted Save.
	if _, statErr := os.Stat(tmp); statErr == nil {
		if v, err := tryDecode[T](tmp); err == nil {
			// Promote tmp to final. If the rename fails we still return the
			// recovered value; the next Save will overwrite the final path.
			_ = os.Rename(tmp, path)
			return v, nil
		}
		// Corrupt tmp: discard and fall through to the final file.
		_ = os.Remove(tmp)
	}

	// Final file missing -> first run, empty state (not an error).
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return zero, nil
	}

	v, err := tryDecode[T](path)
	if err != nil {
		return zero, fmt.Errorf("storage: decode %s: %w", path, err)
	}
	return v, nil
}

// tryDecode reads path and unmarshals it into a new T. A nil error indicates
// success; any I/O or decode failure yields (zero, err).
func tryDecode[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, err
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, err
	}
	return v, nil
}
