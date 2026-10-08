package deduplicator

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func createFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestWalkAndHashExcludes(t *testing.T) {
	hash := func(string) (string, error) { return "h", nil }

	t.Run("default exclusions", func(t *testing.T) {
		dir := t.TempDir()
		createFile(t, filepath.Join(dir, "a.txt"), "same")
		createFile(t, filepath.Join(dir, "b.txt"), "same")
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
			t.Fatalf("mkdir .git: %v", err)
		}
		createFile(t, filepath.Join(dir, ".git", "c.txt"), "same")
		if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0755); err != nil {
			t.Fatalf("mkdir node_modules: %v", err)
		}
		createFile(t, filepath.Join(dir, "node_modules", "d.txt"), "same")

		files, errs := WalkAndHash(dir, []string{".git", "node_modules"}, hash)
		if len(errs) > 0 {
			t.Fatalf("WalkAndHash: %v", errs)
		}
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
		for _, fi := range files {
			base := filepath.Base(fi.Path)
			if base != "a.txt" && base != "b.txt" {
				t.Fatalf("unexpected file %s", fi.Path)
			}
		}
	})

	t.Run("custom exclusion", func(t *testing.T) {
		dir := t.TempDir()
		createFile(t, filepath.Join(dir, "a.txt"), "same")
		createFile(t, filepath.Join(dir, "b.txt"), "same")
		if err := os.Mkdir(filepath.Join(dir, "vendor"), 0755); err != nil {
			t.Fatalf("mkdir vendor: %v", err)
		}
		createFile(t, filepath.Join(dir, "vendor", "c.txt"), "same")

		files, errs := WalkAndHash(dir, []string{".git", "node_modules", "vendor"}, hash)
		if len(errs) > 0 {
			t.Fatalf("WalkAndHash: %v", errs)
		}
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
		for _, fi := range files {
			base := filepath.Base(fi.Path)
			if base != "a.txt" && base != "b.txt" {
				t.Fatalf("unexpected file %s", fi.Path)
			}
		}
	})
}

func TestWalkAndHashCache(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "a.txt"), "same")
	createFile(t, filepath.Join(dir, "b.txt"), "same")

	var count int32
	hash := func(string) (string, error) {
		atomic.AddInt32(&count, 1)
		return "h", nil
	}

	if _, errs := WalkAndHashWithAlgorithm(dir, nil, "sha256", hash); len(errs) > 0 {
		t.Fatalf("WalkAndHash: %v", errs)
	}
	if count != 2 {
		t.Fatalf("expected 2 hash calls, got %d", count)
	}

	if _, errs := WalkAndHashWithAlgorithm(dir, nil, "sha256", hash); len(errs) > 0 {
		t.Fatalf("WalkAndHash second: %v", errs)
	}
	if count != 2 {
		t.Fatalf("expected cache to prevent hashing, got %d calls", count)
	}
}

func TestWalkAndHashCacheUsesAlgorithm(t *testing.T) {
	dir := t.TempDir()
	createFile(t, filepath.Join(dir, "a.txt"), "same")
	createFile(t, filepath.Join(dir, "b.txt"), "same")

	var count int32
	hash := func(path string) (string, error) {
		atomic.AddInt32(&count, 1)
		return filepath.Base(path), nil
	}
	for _, algorithm := range []string{"sha256", "sha1", "sha1"} {
		if _, errs := WalkAndHashWithAlgorithm(dir, nil, algorithm, hash); len(errs) > 0 {
			t.Fatalf("WalkAndHashWithAlgorithm(%s): %v", algorithm, errs)
		}
	}
	if count != 4 {
		t.Fatalf("expected cache reuse only for repeated sha1, got %d hash calls", count)
	}
}

func TestWalkAndHashCacheUsesNanosecondModTime(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	createFile(t, fileA, "same")
	createFile(t, fileB, "same")

	base := time.Unix(1_700_000_000, 100_000_000)
	if err := os.Chtimes(fileA, base, base); err != nil {
		t.Fatalf("set initial file time: %v", err)
	}
	if err := os.Chtimes(fileB, base, base); err != nil {
		t.Fatalf("set initial file time: %v", err)
	}

	var count int32
	hash := func(string) (string, error) {
		atomic.AddInt32(&count, 1)
		return "h", nil
	}
	if _, errs := WalkAndHashWithAlgorithm(dir, nil, "sha256", hash); len(errs) > 0 {
		t.Fatalf("initial scan: %v", errs)
	}

	createFile(t, fileA, "diff")
	updated := base.Add(500 * time.Millisecond)
	if err := os.Chtimes(fileA, updated, updated); err != nil {
		t.Fatalf("set updated file time: %v", err)
	}
	if updated.Unix() != base.Unix() {
		t.Fatal("test timestamps must be in the same second")
	}

	if _, errs := WalkAndHashWithAlgorithm(dir, nil, "sha256", hash); len(errs) > 0 {
		t.Fatalf("second scan: %v", errs)
	}
	if count != 3 {
		t.Fatalf("expected changed file to be rehashed, got %d hash calls", count)
	}
}

func TestWalkAndHashDoesNotCountSymlinkTargetTwice(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a-target.txt")
	createFile(t, target, "same")
	createFile(t, filepath.Join(dir, "b-copy.txt"), "same")
	if err := os.Symlink(target, filepath.Join(dir, "c-link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	files, errs := WalkAndHash(dir, nil, HashFileSHA256)
	if len(errs) > 0 {
		t.Fatalf("WalkAndHash: %v", errs)
	}
	if len(files) != 2 {
		t.Fatalf("expected symlink target to be counted once, got %d files", len(files))
	}
}

func TestWalkAndHashReportsBrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "broken-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, errs := WalkAndHash(dir, nil, HashFileSHA256)
	if len(errs) == 0 {
		t.Fatal("expected an error for the broken symlink")
	}
}
