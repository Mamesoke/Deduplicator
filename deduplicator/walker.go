// walker.go
package deduplicator

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func WalkAndHash(root string, hashFunc func(string) (string, error)) ([]FileInfo, error) {
	var (
		files   []FileInfo
		paths   = make(chan string)
		results = make(chan FileInfo)
		wg      sync.WaitGroup
	)

	workerCount := runtime.NumCPU()
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for path := range paths {
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				hash, err := hashFunc(path)
				if err != nil {
					continue
				}
				results <- FileInfo{
					Path:         path,
					Size:         info.Size(),
					Hash:         hash,
					LastModified: info.ModTime().Unix(),
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		paths <- path
		return nil
	})
	close(paths)

	for fi := range results {
		files = append(files, fi)
	}

	return files, err
}
