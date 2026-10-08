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

// WalkAndHash scans root and hashes files that share a size with at least one
// other file. Caching is disabled because hashFunc has no stable algorithm ID.
func WalkAndHash(root string, excludes []string, hashFunc func(string) (string, error)) ([]FileInfo, []error) {
	return walkAndHash(root, excludes, "", hashFunc)
}

// WalkAndHashWithAlgorithm scans root and caches hashes for the named algorithm.
func WalkAndHashWithAlgorithm(root string, excludes []string, hashAlgorithm string, hashFunc func(string) (string, error)) ([]FileInfo, []error) {
	if hashAlgorithm == "" {
		return nil, []error{fmt.Errorf("hash algorithm must not be empty")}
	}
	return walkAndHash(root, excludes, hashAlgorithm, hashFunc)
}

func walkAndHash(root string, excludes []string, hashAlgorithm string, hashFunc func(string) (string, error)) ([]FileInfo, []error) {
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

	cacheEnabled := hashAlgorithm != ""
	var cache map[string]CacheEntry
	var err error
	cachePath := ""
	if cacheEnabled {
		root, err = filepath.Abs(root)
		if err != nil {
			return nil, []error{fmt.Errorf("resolving scan root: %w", err)}
		}
		cachePath = filepath.Join(root, cacheFileName)
		cache, err = loadCache(cachePath)
	}
	var (
		cacheMu sync.RWMutex
		errs    []error
		errsMu  sync.Mutex
	)
	if cacheEnabled && err != nil {
		log.Printf("error loading cache: %v", err)
		errs = append(errs, fmt.Errorf("loading cache: %w", err))
		cache = make(map[string]CacheEntry)
	}
	if cacheEnabled && cache == nil {
		cache = make(map[string]CacheEntry)
	}

	type job struct {
		path    string
		size    int64
		modTime int64
	}

	// Primer paso: construir un mapa size -> []job mediante escaneo manual
	sizeMap := make(map[int64][]job)
	var (
		sizeMu    sync.Mutex
		seenFiles = make(map[string]struct{})
	)
	addError := func(err error) {
		errsMu.Lock()
		errs = append(errs, err)
		errsMu.Unlock()
	}
	addFile := func(path, name string) {
		if isExcluded(name) {
			return
		}
		info, err := os.Stat(path)
		if err != nil {
			addError(fmt.Errorf("stat %s: %w", path, err))
			return
		}
		if info.IsDir() {
			return
		}
		canonicalPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			addError(fmt.Errorf("resolving %s: %w", path, err))
			return
		}
		canonicalPath, err = filepath.Abs(canonicalPath)
		if err != nil {
			addError(fmt.Errorf("resolving absolute path for %s: %w", path, err))
			return
		}
		canonicalPath = filepath.Clean(canonicalPath)
		sizeMu.Lock()
		defer sizeMu.Unlock()
		if _, exists := seenFiles[canonicalPath]; exists {
			return
		}
		seenFiles[canonicalPath] = struct{}{}
		sizeMap[info.Size()] = append(sizeMap[info.Size()], job{
			path:    canonicalPath,
			size:    info.Size(),
			modTime: info.ModTime().UnixNano(),
		})
	}

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
						addFile(path, name)
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
					addFile(path, name)
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
				var ce CacheEntry
				var ok bool
				if cacheEnabled {
					cacheMu.RLock()
					ce, ok = cache[j.path]
					cacheMu.RUnlock()
				}
				var hash string
				if ok && ce.Size == j.size && ce.ModTime == j.modTime && ce.HashAlgorithm == hashAlgorithm {
					hash = ce.Hash
				} else {
					var err error
					hash, err = hashFunc(j.path)
					if err != nil {
						log.Printf("error hashing %s: %v", j.path, err)
						addError(fmt.Errorf("hashing %s: %w", j.path, err))
						continue
					}
					if cacheEnabled {
						cacheMu.Lock()
						cache[j.path] = CacheEntry{
							Path:          j.path,
							Size:          j.size,
							ModTime:       j.modTime,
							HashAlgorithm: hashAlgorithm,
							Hash:          hash,
						}
						cacheMu.Unlock()
					}
				}
				results <- FileInfo{
					Path:         j.path,
					Size:         j.size,
					Hash:         hash,
					LastModified: time.Unix(0, j.modTime).Unix(),
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

	if cacheEnabled {
		if err := saveCache(cachePath, cache); err != nil {
			log.Printf("error saving cache: %v", err)
			errs = append(errs, fmt.Errorf("saving cache: %w", err))
		}
	}

	return files, errs
}
