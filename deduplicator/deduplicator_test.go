package deduplicator

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// collectDuplicates is a helper that wraps WalkAndHash and FindDuplicates.
func collectDuplicates(
	walk func(string, []string, func(string) (string, error)) ([]FileInfo, error),
	hash func(string) (string, error),
) ([]DuplicateGroup, error) {
	files, err := walk("root", nil, hash)
	if err != nil {
		return nil, err
	}
	return FindDuplicates(files), nil
}

func TestFindDuplicates(t *testing.T) {
	files := []FileInfo{
		{Path: "a.txt", Size: 1, Hash: "h1"},
		{Path: "b.txt", Size: 1, Hash: "h1"},
		{Path: "c.txt", Size: 2, Hash: "h2"},
	}
	groups := FindDuplicates(files)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Hash != "h1" {
		t.Fatalf("expected hash h1, got %s", groups[0].Hash)
	}
	if len(groups[0].Files) != 2 {
		t.Fatalf("expected 2 files in group, got %d", len(groups[0].Files))
	}
}

func TestCollectDuplicatesSuccess(t *testing.T) {
	mockWalk := func(root string, excludes []string, h func(string) (string, error)) ([]FileInfo, error) {
		return []FileInfo{
			{Path: "a", Size: 1, Hash: "same"},
			{Path: "b", Size: 1, Hash: "same"},
		}, nil
	}
	mockHash := func(path string) (string, error) { return "same", nil }

	groups, err := collectDuplicates(mockWalk, mockHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 || len(groups[0].Files) != 2 {
		t.Fatalf("unexpected groups: %#v", groups)
	}
}

func TestCollectDuplicatesWalkError(t *testing.T) {
	walkErr := errors.New("walk error")
	mockWalk := func(root string, excludes []string, h func(string) (string, error)) ([]FileInfo, error) {
		return nil, walkErr
	}
	mockHash := func(path string) (string, error) { return "", nil }

	if _, err := collectDuplicates(mockWalk, mockHash); err != walkErr {
		t.Fatalf("expected %v, got %v", walkErr, err)
	}
}

func TestWalkAndHashHashError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("write good: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("write bad: %v", err)
	}

	mockHash := func(path string) (string, error) {
		if filepath.Base(path) == "bad.txt" {
			return "", errors.New("hash fail")
		}
		return "ok", nil
	}

	files, err := WalkAndHash(dir, nil, mockHash)
	if err != nil {
		t.Fatalf("WalkAndHash error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if filepath.Base(files[0].Path) != "good.txt" {
		t.Fatalf("unexpected file: %s", files[0].Path)
	}
}
