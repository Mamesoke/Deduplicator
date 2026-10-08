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
	dir := flag.String("dir", "", "Ruta del directorio a analizar")
	format := flag.String("format", "pretty", "Formato de salida: pretty | json")
	hashAlg := flag.String("hash", "sha256", "Algoritmo de hash: sha256 | sha1 | sha512 | md5")
	deleteFlag := flag.Bool("delete", false, "Eliminar automáticamente los archivos duplicados")
	dryRun := flag.Bool("dry-run", false, "Simular la eliminación sin borrar archivos")
	timings := flag.Bool("timings", false, "Mostrar duración de cada goroutine")
	excludes := multiFlag{".git", "node_modules", ".github", ".idea", ".vscode", "vendor", "dist", "build", "tmp", "temp", ".venv", "venv"}
	flag.Var(&excludes, "exclude", "Patrones o rutas a excluir (puede usarse varias veces)")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "Uso: dedup-cli -dir=/ruta/a/analizar")
		os.Exit(1)
	}
	if *format != "pretty" && *format != "json" {
		fmt.Fprintf(os.Stderr, "Formato no reconocido: %s\n", *format)
		os.Exit(1)
	}

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
		fmt.Fprintf(os.Stderr, "Algoritmo de hash no soportado: %s\n", *hashAlg)
		os.Exit(1)
	}

	log.Printf("Analizando directorio: %s", *dir)
	if *timings {
		deduplicator.MeasureTimings = true
	}

	files, errs := deduplicator.WalkAndHashWithAlgorithm(*dir, []string(excludes), *hashAlg, hashFunc)
	exitCode := 0
	for _, err := range errs {
		log.Printf("Error durante el escaneo: %v", err)
		exitCode = 1
	}

	dupes := deduplicator.FindDuplicates(files)
	if *format == "json" {
		if err := deduplicator.JSONPrintTo(os.Stdout, dupes); err != nil {
			log.Printf("Error al generar salida JSON: %v", err)
			os.Exit(1)
		}
	} else if len(dupes) == 0 {
		fmt.Println("No se encontraron duplicados.")
	} else {
		deduplicator.PrettyPrint(dupes)
	}

	if *deleteFlag {
		diagnostics := os.Stdout
		if *format == "json" {
			diagnostics = os.Stderr
		}
		removed, err := deduplicator.DeleteDuplicatesTo(diagnostics, dupes, *dryRun)
		if err != nil {
			log.Printf("Error al eliminar duplicados: %v", err)
			exitCode = 1
		}
		if *dryRun {
			fmt.Fprintf(diagnostics, "Se eliminarían %d archivos duplicados.\n", len(removed))
		} else {
			fmt.Fprintf(diagnostics, "Se eliminaron %d archivos duplicados.\n", len(removed))
		}
	}

	os.Exit(exitCode)
}
