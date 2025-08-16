// types.go
package deduplicator

type FileInfo struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	Hash         string `json:"hash"`
	LastModified int64  `json:"lastModified"`
}

type DuplicateGroup struct {
	Hash  string     `json:"hash"`
	Files []FileInfo `json:"files"`
}

// CacheEntry represents a file entry stored in the cache.
// It is keyed by the absolute file path and stores the size,
// modification time and the previously computed hash so we can
// avoid recalculating hashes for unchanged files.
type CacheEntry struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	Hash    string `json:"hash"`
}
