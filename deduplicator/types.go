// types.go
package deduplicator

type FileInfo struct {
	Path         string
	Size         int64
	Hash         string
	LastModified int64
}

type DuplicateGroup struct {
	Hash      string
	Files     []FileInfo
}