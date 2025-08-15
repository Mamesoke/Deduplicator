package deduplicator

import (
	"encoding/json"
	"fmt"
	"sort"
)

type DuplicateReport struct {
	Groups      []DuplicateGroup `json:"groups"`
	TotalFiles  int              `json:"total_duplicated_files"`
	TotalWasted int64            `json:"total_wasted_bytes"`
	TotalGroups int              `json:"total_groups"`
}

// PrettyPrint imprime el resultado formateado para humanos
func PrettyPrint(dupes []DuplicateGroup) {
	totalDuplicatedFiles := 0
	totalWastedBytes := int64(0)

	for i, group := range dupes {
		numFiles := len(group.Files)
		if numFiles <= 1 {
			continue
		}

		sizePerFile := group.Files[0].Size
		wasted := int64(numFiles-1) * sizePerFile
		totalDuplicatedFiles += numFiles - 1
		totalWastedBytes += wasted

		fmt.Printf("🔁 Grupo #%d — %d archivos duplicados (Hash: %s)\n", i+1, numFiles, group.Hash)
		fmt.Printf("    Tamaño por archivo: %d bytes | Total duplicado: %d bytes\n", sizePerFile, wasted)

		sorted := append([]FileInfo(nil), group.Files...)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Path < sorted[j].Path
		})

		for _, f := range sorted {
			fmt.Printf("    - %s\n", f.Path)
		}
		fmt.Println()
	}

	fmt.Println("📊 Resumen:")
	fmt.Printf("  - Total de grupos de duplicados: %d\n", len(dupes))
	fmt.Printf("  - Total de archivos duplicados: %d\n", totalDuplicatedFiles)
	fmt.Printf("  - Espacio potencial recuperable: %.2f MB\n", float64(totalWastedBytes)/1024.0/1024.0)
}

// JSONPrint imprime el resultado en JSON estructurado
func JSONPrint(dupes []DuplicateGroup) error {
	// Ordenar los grupos por desperdicio descendente sin mutar la entrada original
	sorted := make([]DuplicateGroup, len(dupes))
	copy(sorted, dupes)
	sort.Slice(sorted, func(i, j int) bool {
		wasteI := int64(len(sorted[i].Files)-1) * sorted[i].Files[0].Size
		wasteJ := int64(len(sorted[j].Files)-1) * sorted[j].Files[0].Size
		return wasteI > wasteJ
	})

	// Clonar y ordenar los archivos de cada grupo por su ruta
	for i, group := range sorted {
		files := append([]FileInfo(nil), group.Files...)
		sort.Slice(files, func(a, b int) bool {
			return files[a].Path < files[b].Path
		})
		sorted[i].Files = files
	}

	report := DuplicateReport{
		Groups:      sorted,
		TotalGroups: len(sorted),
	}

	for _, group := range sorted {
		n := len(group.Files)
		if n > 1 {
			report.TotalFiles += n - 1
			report.TotalWasted += int64(n-1) * group.Files[0].Size
		}
	}

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}
