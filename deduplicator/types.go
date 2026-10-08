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

// CacheEntry represents a file entry stored in the cache. It is keyed by
// absolute file path and includes the hash algorithm so cached values are
// never reused across algorithms. ModTime is stored as Unix nanoseconds.
type CacheEntry struct {
	Path          string `json:"path"`
	Size          int64  `json:"size"`
	ModTime       int64  `json:"modTime"`
	HashAlgorithm string `json:"hashAlgorithm"`
	Hash          string `json:"hash"`
}
