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
