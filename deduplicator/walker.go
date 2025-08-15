// walker.go
package deduplicator

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// WalkAndHash recorre el directorio, agrupa primero por tamaño y solo
// calcula el hash de los archivos que comparten tamaño con al menos otro.
func WalkAndHash(root string, excludes []string, hashFunc func(string) (string, error)) ([]FileInfo, error) {
	isExcluded := func(path string) bool {
		for _, pattern := range excludes {
			if ok, _ := filepath.Match(pattern, path); ok {
				return true
			}
			if ok, _ := filepath.Match(pattern, filepath.Base(path)); ok {
				return true
			}
		}
		return false
	}

	type job struct {
		path    string
		size    int64
		modTime int64
	}

	// Primer paso: construir un mapa size -> []job
	sizeMap := make(map[int64][]job)
	visited := make(map[string]struct{})
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}
			target, err = filepath.Abs(target)
			if err != nil {
				return nil
			}
			target = filepath.Clean(target)
			if _, ok := visited[target]; ok {
				return nil
			}
			visited[target] = struct{}{}
			if isExcluded(target) {
				return nil
			}
			info, err := os.Stat(target)
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			sizeMap[info.Size()] = append(sizeMap[info.Size()], job{
				path:    target,
				size:    info.Size(),
				modTime: info.ModTime().Unix(),
			})
			return nil
		}
		if d.IsDir() {
			if isExcluded(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if isExcluded(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		sizeMap[info.Size()] = append(sizeMap[info.Size()], job{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime().Unix(),
		})
		return nil
	}); err != nil {
		return nil, err
	}

	var (
		files   []FileInfo
		paths   = make(chan job)
		results = make(chan FileInfo)
		wg      sync.WaitGroup
	)

	workerCount := runtime.NumCPU()
	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for j := range paths {
				hash, err := hashFunc(j.path)
				if err != nil {
					log.Printf("error hashing %s: %v", j.path, err)
					continue
				}
				results <- FileInfo{
					Path:         j.path,
					Size:         j.size,
					Hash:         hash,
					LastModified: j.modTime,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, group := range sizeMap {
			if len(group) > 1 {
				for _, j := range group {
					if isExcluded(j.path) {
						continue
					}
					paths <- j
				}
			}
		}
		close(paths)
	}()

	for fi := range results {
		files = append(files, fi)
	}

	return files, nil
}
