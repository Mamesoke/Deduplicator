package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"deduplicator/deduplicator"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return fmt.Sprint([]string(*m))
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	// Flags de entrada
	dir := flag.String("dir", "", "Ruta del directorio a analizar")
	format := flag.String("format", "pretty", "Formato de salida: pretty | json")
	hashAlg := flag.String("hash", "sha256", "Algoritmo de hash: sha256 | sha1 | sha512 | md5")
	deleteFlag := flag.Bool("delete", false, "Eliminar automáticamente los archivos duplicados")
	dryRun := flag.Bool("dry-run", false, "Simular la eliminación sin borrar archivos")
	var excludes = multiFlag{".git", "node_modules"}
	flag.Var(&excludes, "exclude", "Patrones o rutas a excluir (puede usarse varias veces)")
	flag.Parse()

	if *dir == "" {
		fmt.Println("Uso: dedup-cli -dir=/ruta/a/analizar")
		os.Exit(1)
	}

	fmt.Printf("Analizando directorio: %s\n", *dir)

	var hashFunc func(string) (string, error)
	switch *hashAlg {
	case "sha256":
		hashFunc = deduplicator.HashFileSHA256
	case "sha1":
		hashFunc = deduplicator.HashFileSHA1
	case "sha512":
		hashFunc = deduplicator.HashFileSHA512
	case "md5":
		hashFunc = deduplicator.HashFileMD5
	default:
		fmt.Printf("Algoritmo de hash no soportado: %s\n", *hashAlg)
		os.Exit(1)
	}

	// Escanear archivos y calcular hashes
	files, err := deduplicator.WalkAndHash(*dir, []string(excludes), hashFunc)
	if err != nil {
		log.Fatalf("Error durante el escaneo: %v", err)
	}

	// Buscar duplicados
	dupes := deduplicator.FindDuplicates(files)

	if len(dupes) == 0 {
		fmt.Println("No se encontraron duplicados.")
		return
	}
	/*
	   // Mostrar resultados
	   fmt.Printf("Se encontraron %d grupos de duplicados:\n\n", len(dupes))
	   for i, group := range dupes {
	   fmt.Printf("Grupo #%d (Hash: %s)\n", i+1, group.Hash)
	   for _, f := range group.Files {
	   fmt.Printf("  - %s (%d bytes)\n", f.Path, f.Size)
	   }
	   fmt.Println()
	   }
	*/
	switch *format {
	case "json":
		if err := deduplicator.JSONPrint(dupes); err != nil {
			log.Fatalf("Error al generar salida JSON: %v", err)
		}
	case "pretty":
		deduplicator.PrettyPrint(dupes)
	default:
		fmt.Printf("Formato no reconocido: %s\n", *format)
		os.Exit(1)
	}

	if *deleteFlag {
		removed, err := deduplicator.DeleteDuplicates(dupes, *dryRun)
		if err != nil {
			log.Fatalf("Error al eliminar duplicados: %v", err)
		}
		if *dryRun {
			fmt.Printf("Se eliminarían %d archivos duplicados.\n", len(removed))
		} else {
			fmt.Printf("Se eliminaron %d archivos duplicados.\n", len(removed))
		}
	}
	// Mostrar resultados mejorados
	/*
	   totalDuplicatedFiles := 0
	   totalWastedBytes := int64(0)

	   for i, group := range dupes {
	   numFiles := len(group.Files)
	   if numFiles <= 1 {
	   continue // debería ser innecesario, pero por seguridad
	   }

	   sizePerFile := group.Files[0].Size
	   wasted := int64(numFiles-1) * sizePerFile
	   totalDuplicatedFiles += numFiles - 1
	   totalWastedBytes += wasted

	   fmt.Printf("🔁 Grupo #%d — %d archivos duplicados (Hash: %s)\n", i+1, numFiles, group.Hash)
	   fmt.Printf("    Tamaño por archivo: %d bytes | Total duplicado: %d bytes\n", sizePerFile, wasted)

	   // Ordenar las rutas para facilitar lectura
	   sorted := group.Files
	   sort.Slice(sorted, func(i, j int) bool {
	   return sorted[i].Path < sorted[j].Path
	   })

	   for _, f := range sorted {
	   fmt.Printf("    - %s\n", f.Path)
	   }
	   fmt.Println()
	   }

	   // Resumen final
	   fmt.Println("📊 Resumen:")
	   fmt.Printf("  - Total de grupos de duplicados: %d\n", len(dupes))
	   fmt.Printf("  - Total de archivos duplicados: %d\n", totalDuplicatedFiles)
	   fmt.Printf("  - Espacio potencial recuperable: %.2f MB\n", float64(totalWastedBytes)/1024.0/1024.0)
	*/
}
