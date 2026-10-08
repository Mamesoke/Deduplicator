package deduplicator

import (
	"bytes"
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

func TestDeleteDuplicatesRefusesDifferentContents(t *testing.T) {
	dir := t.TempDir()
	keeper := filepath.Join(dir, "keeper")
	candidate := filepath.Join(dir, "candidate")
	if err := os.WriteFile(keeper, []byte("safe"), 0o644); err != nil {
		t.Fatalf("write keeper: %v", err)
	}
	if err := os.WriteFile(candidate, []byte("evil"), 0o644); err != nil {
		t.Fatalf("write candidate: %v", err)
	}
	groups := []DuplicateGroup{{
		Hash:  "colliding-hash",
		Files: []FileInfo{{Path: keeper}, {Path: candidate}},
	}}

	removed, err := DeleteDuplicates(groups, false)
	if err == nil {
		t.Fatal("expected deletion to be refused for different contents")
	}
	if len(removed) != 0 {
		t.Fatalf("expected no files to be reported as removed, got %v", removed)
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("candidate should not have been deleted: %v", err)
	}
}

func TestDeleteDuplicatesRefusesSamePathTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	groups := []DuplicateGroup{{Files: []FileInfo{{Path: path}, {Path: path}}}}

	if _, err := DeleteDuplicates(groups, false); err == nil {
		t.Fatal("expected duplicate path to be refused")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("kept file should still exist: %v", err)
	}
}

func TestDeleteDuplicatesToWritesDryRunToProvidedWriter(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	for _, path := range []string{a, b} {
		if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	var output bytes.Buffer
	groups := []DuplicateGroup{{Files: []FileInfo{{Path: a}, {Path: b}}}}
	removed, err := DeleteDuplicatesTo(&output, groups, true)
	if err != nil {
		t.Fatalf("DeleteDuplicatesTo: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("expected one planned removal, got %d", len(removed))
	}
	if !bytes.Contains(output.Bytes(), []byte(b)) {
		t.Fatalf("expected dry-run output to include %q, got %q", b, output.String())
	}
	if _, err := os.Stat(b); err != nil {
		t.Fatalf("dry-run should not delete candidate: %v", err)
	}
}
