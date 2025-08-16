package deduplicator

import (
	"encoding/json"
	"os"
)

// loadCache reads the cache file and returns the cache map. If the file does
// not exist it returns an empty cache.
func loadCache(path string) (map[string]CacheEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]CacheEntry), nil
		}
		return nil, err
	}
	defer f.Close()

	var cache map[string]CacheEntry
	if err := json.NewDecoder(f).Decode(&cache); err != nil {
		return nil, err
	}
	return cache, nil
}

// saveCache writes the cache map to the given path in JSON format.
func saveCache(path string, cache map[string]CacheEntry) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cache); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
