// walker.go
package deduplicator

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const cacheFileName = ".dedupcache.json"

// WalkAndHash recorre el directorio, agrupa primero por tamaño y solo
// calcula el hash de los archivos que comparten tamaño con al menos otro.
func WalkAndHash(root string, excludes []string, hashFunc func(string) (string, error)) ([]FileInfo, []error) {
	isExcluded := func(name string) bool {
		if name == cacheFileName {
			return true
		}
		for _, pattern := range excludes {
			if ok, _ := filepath.Match(pattern, name); ok {
				return true
			}
		}
		return false
	}

	cachePath := filepath.Join(root, cacheFileName)
	cache, err := loadCache(cachePath)
	var (
		cacheMu sync.RWMutex
		errs    []error
		errsMu  sync.Mutex
	)
	if err != nil {
		log.Printf("error loading cache: %v", err)
		errs = append(errs, fmt.Errorf("loading cache: %w", err))
		cache = make(map[string]CacheEntry)
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

	dirs := make(chan string, runtime.NumCPU()*4)
	var dirWG sync.WaitGroup
	var workerWG sync.WaitGroup

	scanWorkers := runtime.NumCPU()
	workerWG.Add(scanWorkers)
	for i := 0; i < scanWorkers; i++ {
		go func(id int) {
			start := time.Now()
			defer func() {
				if MeasureTimings {
					log.Printf("scan worker %d took %v", id, time.Since(start))
				}
				workerWG.Done()
			}()
			for dir := range dirs {
				entries, err := os.ReadDir(dir)
				if err != nil {
					log.Printf("error reading %s: %v", dir, err)
					errsMu.Lock()
					errs = append(errs, fmt.Errorf("reading %s: %w", dir, err))
					errsMu.Unlock()
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
						// Avoid blocking all workers when channel is full:
						// try to send, but fall back to a goroutine if the queue is saturated.
						select {
						case dirs <- path:
						default:
							go func(p string) { dirs <- p }(path)
						}
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
		}(i)
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
		go func(id int) {
			start := time.Now()
			defer func() {
				if MeasureTimings {
					log.Printf("hash worker %d took %v", id, time.Since(start))
				}
				wg.Done()
			}()
			for j := range paths {
				cacheMu.RLock()
				ce, ok := cache[j.path]
				cacheMu.RUnlock()
				var hash string
				if ok && ce.Size == j.size && ce.ModTime == j.modTime {
					hash = ce.Hash
				} else {
					var err error
					hash, err = hashFunc(j.path)
					if err != nil {
						log.Printf("error hashing %s: %v", j.path, err)
						errsMu.Lock()
						errs = append(errs, fmt.Errorf("hashing %s: %w", j.path, err))
						errsMu.Unlock()
						continue
					}
					cacheMu.Lock()
					cache[j.path] = CacheEntry{
						Path:    j.path,
						Size:    j.size,
						ModTime: j.modTime,
						Hash:    hash,
					}
					cacheMu.Unlock()
				}
				results <- FileInfo{
					Path:         j.path,
					Size:         j.size,
					Hash:         hash,
					LastModified: j.modTime,
				}
			}
		}(i)
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

	if err := saveCache(cachePath, cache); err != nil {
		log.Printf("error saving cache: %v", err)
		errs = append(errs, fmt.Errorf("saving cache: %w", err))
	}

	return files, errs
}
