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
	isExcluded := func(name string) bool {
		for _, pattern := range excludes {
			if ok, _ := filepath.Match(pattern, name); ok {
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

	// Primer paso: construir un mapa size -> []job mediante escaneo manual
	sizeMap := make(map[int64][]job)
	visited := make(map[string]struct{})
	var (
		sizeMu    sync.Mutex
		visitedMu sync.Mutex
	)

	dirs := make(chan string)
	var dirWG sync.WaitGroup
	var workerWG sync.WaitGroup

	scanWorkers := runtime.NumCPU()
	workerWG.Add(scanWorkers)
	for i := 0; i < scanWorkers; i++ {
		go func() {
			defer workerWG.Done()
			for dir := range dirs {
				entries, err := os.ReadDir(dir)
				if err != nil {
					log.Printf("error reading %s: %v", dir, err)
					dirWG.Done()
					continue
				}
				for _, entry := range entries {
					name := entry.Name()
					path := filepath.Join(dir, name)
					if entry.Type()&fs.ModeSymlink != 0 {
						target, err := os.Readlink(path)
						if err != nil {
							continue
						}
						if !filepath.IsAbs(target) {
							target = filepath.Join(dir, target)
						}
						target, err = filepath.Abs(target)
						if err != nil {
							continue
						}
						target = filepath.Clean(target)
						visitedMu.Lock()
						if _, ok := visited[target]; ok {
							visitedMu.Unlock()
							continue
						}
						visited[target] = struct{}{}
						visitedMu.Unlock()
						if isExcluded(filepath.Base(target)) {
							continue
						}
						info, err := os.Stat(target)
						if err != nil || info.IsDir() {
							continue
						}
						sizeMu.Lock()
						sizeMap[info.Size()] = append(sizeMap[info.Size()], job{
							path:    target,
							size:    info.Size(),
							modTime: info.ModTime().Unix(),
						})
						sizeMu.Unlock()
						continue
					}
					if entry.IsDir() {
						if isExcluded(name) {
							continue
						}
						dirWG.Add(1)
						dirs <- path
						continue
					}
					if isExcluded(name) {
						continue
					}
					info, err := entry.Info()
					if err != nil {
						continue
					}
					sizeMu.Lock()
					sizeMap[info.Size()] = append(sizeMap[info.Size()], job{
						path:    path,
						size:    info.Size(),
						modTime: info.ModTime().Unix(),
					})
					sizeMu.Unlock()
				}
				dirWG.Done()
			}
		}()
	}

	dirWG.Add(1)
	dirs <- root
	go func() {
		dirWG.Wait()
		close(dirs)
	}()
	workerWG.Wait()

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
					if isExcluded(filepath.Base(j.path)) {
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
