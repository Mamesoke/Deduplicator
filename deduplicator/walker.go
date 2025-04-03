// walker.go
package deduplicator

import (
	"io/fs"
	"os"
	"path/filepath"
)

func WalkAndHash(root string, hashFunc func(string) (string, error)) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // ignorar errores o carpetas
		}

		info, _ := os.Stat(path)
		hash, err := hashFunc(path)
		if err != nil {
			return nil // opcional: loggear errores por archivo
		}

		files = append(files, FileInfo{
			Path:         path,
			Size:         info.Size(),
			Hash:         hash,
			LastModified: info.ModTime().Unix(),
		})

		return nil
	})

	return files, err
}
