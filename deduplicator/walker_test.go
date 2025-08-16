package deduplicator

import (
	"os"
	"path/filepath"
	"testing"
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

		files, err := WalkAndHash(dir, []string{".git", "node_modules"}, hash)
		if err != nil {
			t.Fatalf("WalkAndHash: %v", err)
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

		files, err := WalkAndHash(dir, []string{".git", "node_modules", "vendor"}, hash)
		if err != nil {
			t.Fatalf("WalkAndHash: %v", err)
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
