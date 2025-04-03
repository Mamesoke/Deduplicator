// deduplicator.go
package deduplicator

func FindDuplicates(files []FileInfo) []DuplicateGroup {
	hashMap := make(map[string][]FileInfo)

	for _, f := range files {
		hashMap[f.Hash] = append(hashMap[f.Hash], f)
	}

	var duplicates []DuplicateGroup
	for hash, group := range hashMap {
		if len(group) > 1 {
			duplicates = append(duplicates, DuplicateGroup{
				Hash:  hash,
				Files: group,
			})
		}
	}
	return duplicates
}
