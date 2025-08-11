// deduplicator.go
package deduplicator

import "sort"

func FindDuplicates(files []FileInfo) []DuplicateGroup {
	hashMap := make(map[string][]FileInfo)

	for _, f := range files {
		hashMap[f.Hash] = append(hashMap[f.Hash], f)
	}

	keys := make([]string, 0, len(hashMap))
	for k := range hashMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var duplicates []DuplicateGroup
	for _, hash := range keys {
		group := hashMap[hash]
		if len(group) > 1 {
			duplicates = append(duplicates, DuplicateGroup{
				Hash:  hash,
				Files: group,
			})
		}
	}
	return duplicates
}
