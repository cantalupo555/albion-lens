package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type sample struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	in := sample{Name: "fame", Value: 12345}
	if err := Save(path, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load[sample](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out != in {
		t.Errorf("round-trip = %+v, want %+v", out, in)
	}
}

func TestSaveLeavesNoTmpAndCreatesFinal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	if err := Save(path, sample{Name: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(tmpPath(path)); !os.IsNotExist(err) {
		t.Errorf("tmp file should not exist after Save; stat err=%v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("final file should exist after Save; stat err=%v", err)
	}
}

func TestLoadMissingFileReturnsZeroNoError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.json")

	v, err := Load[sample](path)
	if err != nil {
		t.Errorf("Load missing file returned err: %v", err)
	}
	if v != (sample{}) {
		t.Errorf("Load missing = %+v, want zero value", v)
	}
}

func TestLoadCorruptFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v, err := Load[sample](path)
	if err == nil {
		t.Fatalf("Load corrupt file should return error, got nil with value %+v", v)
	}
	if v != (sample{}) {
		t.Errorf("Load corrupt = %+v, want zero value", v)
	}
}

func TestLoadPromotesValidTmp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	tmp := tmpPath(path)

	// Only the tmp file exists — simulating a Save that wrote tmp but crashed
	// before the rename.
	good := sample{Name: "recovered", Value: 99}
	data, err := json.Marshal(good)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v, err := Load[sample](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v != good {
		t.Errorf("Load = %+v, want promoted %+v", v, good)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("tmp should be gone after promotion; stat err=%v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("final should exist after promotion; stat err=%v", err)
	}
}

func TestLoadDiscardsCorruptTmpAndUsesFinal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	// Corrupt tmp + valid final: the corrupt tmp must be discarded and the
	// final file used.
	if err := os.WriteFile(tmpPath(path), []byte("garbage"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	good := sample{Name: "final", Value: 7}
	data, err := json.Marshal(good)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v, err := Load[sample](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v != good {
		t.Errorf("Load = %+v, want final %+v", v, good)
	}
	if _, err := os.Stat(tmpPath(path)); !os.IsNotExist(err) {
		t.Errorf("corrupt tmp should have been removed; stat err=%v", err)
	}
}

func TestLoadIgnoresUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	// Forward-compat: JSON with extra fields not present in the sample struct
	// decodes successfully (encoding/json ignores unknown fields by default).
	raw := []byte(`{"name":"x","value":1,"futureField":"ignore-me","extra":42}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v, err := Load[sample](path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v.Name != "x" || v.Value != 1 {
		t.Errorf("Load = %+v, want parsed sample", v)
	}
}

func TestSaveMarshalErrorPropagates(t *testing.T) {
	// json.Marshal fails on unsupported types (chan/func). This deterministically
	// exercises the marshal-error path of Save without any OS/filesystem tricks.
	path := filepath.Join(t.TempDir(), "bad.json")
	err := Save(path, map[string]any{"chan": make(chan int)})
	if err == nil {
		t.Fatal("expected marshal error for unsupported type, got nil")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("final file must not be written when marshal fails; stat err=%v", statErr)
	}
}

func TestSaveRenameFailureCleansUpTmp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	tmp := tmpPath(path)

	// Make the final path a non-empty directory so os.Rename(tmp, path) fails
	// (rename cannot overwrite a non-empty directory). This forces the cleanup
	// branch in Save.
	if err := os.MkdirAll(filepath.Join(path, "blocker"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	err := Save(path, sample{Name: "x", Value: 1})
	if err == nil {
		t.Fatal("expected Rename error, got nil")
	}
	if _, statErr := os.Stat(tmp); !os.IsNotExist(statErr) {
		t.Errorf("tmp should be removed after rename failure; stat err=%v", statErr)
	}
}

func TestLoadRenameFailureStillReturnsRecoveredValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	tmp := tmpPath(path)

	// A valid leftover .tmp (interrupted Save) plus a final path that is a
	// non-empty directory: promotion via os.Rename fails, but Load must still
	// return the recovered value (best-effort, documented behavior).
	good := sample{Name: "recovered", Value: 42}
	data, err := json.Marshal(good)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(path, "blocker"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	v, err := Load[sample](path)
	if err != nil {
		t.Fatalf("Load should return recovered value even if rename fails: %v", err)
	}
	if v != good {
		t.Errorf("Load = %+v, want recovered %+v", v, good)
	}
}

func TestConcurrentSaveSamePathStaysConsistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			if err := Save(path, sample{Name: "x", Value: i}); err != nil {
				t.Errorf("Save %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// The final file must be readable and valid regardless of which writer
	// won the race.
	v, err := Load[sample](path)
	if err != nil {
		t.Fatalf("final Load: %v", err)
	}
	if v.Name != "x" {
		t.Errorf("final = %+v, want name %q", v, "x")
	}
	// No leftover tmp after the dust settles.
	if _, err := os.Stat(tmpPath(path)); !os.IsNotExist(err) {
		t.Errorf("tmp should not exist after concurrent saves; stat err=%v", err)
	}
}
