package deduplicator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteDuplicates(t *testing.T) {
	dir := t.TempDir()

	g1a := filepath.Join(dir, "g1a")
	g1b := filepath.Join(dir, "g1b")
	g1c := filepath.Join(dir, "g1c")
	for _, p := range []string{g1a, g1b, g1c} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	g2a := filepath.Join(dir, "g2a")
	g2b := filepath.Join(dir, "g2b")
	for _, p := range []string{g2a, g2b} {
		if err := os.WriteFile(p, []byte("y"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	groups := []DuplicateGroup{
		{Hash: "h1", Files: []FileInfo{{Path: g1a}, {Path: g1b}, {Path: g1c}}},
		{Hash: "h2", Files: []FileInfo{{Path: g2a}, {Path: g2b}}},
	}

	removed, err := DeleteDuplicates(groups, false)
	if err != nil {
		t.Fatalf("DeleteDuplicates: %v", err)
	}
	if len(removed) != 3 {
		t.Fatalf("expected 3 deletions, got %d", len(removed))
	}

	if _, err := os.Stat(g1a); err != nil {
		t.Fatalf("kept file missing: %v", err)
	}
	if _, err := os.Stat(g2a); err != nil {
		t.Fatalf("kept file missing: %v", err)
	}
	for _, p := range []string{g1b, g1c, g2b} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s was not deleted", p)
		}
	}
}

func TestDeleteDuplicatesDryRun(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	groups := []DuplicateGroup{{Hash: "h", Files: []FileInfo{{Path: a}, {Path: b}}}}

	removed, err := DeleteDuplicates(groups, true)
	if err != nil {
		t.Fatalf("DeleteDuplicates dry-run: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removal in dry-run, got %d", len(removed))
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("file %s should exist: %v", p, err)
		}
	}
}
