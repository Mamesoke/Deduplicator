package deduplicator

import (
	"fmt"
	"os"
)

// DeleteDuplicates elimina las copias redundantes de cada grupo dejando
// el primer archivo intacto. Devuelve las rutas que fueron (o serían)
// eliminadas.
func DeleteDuplicates(groups []DuplicateGroup, dryRun bool) ([]string, error) {
	var removed []string
	for _, g := range groups {
		for i := 1; i < len(g.Files); i++ {
			path := g.Files[i].Path
			removed = append(removed, path)
			if dryRun {
				fmt.Printf("[dry-run] eliminar %s\n", path)
				continue
			}
			if err := os.Remove(path); err != nil {
				return removed, err
			}
		}
	}
	return removed, nil
}
