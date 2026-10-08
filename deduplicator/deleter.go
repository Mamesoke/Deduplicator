package deduplicator

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// DeleteDuplicates elimina las copias redundantes de cada grupo dejando
// el primer archivo intacto. Devuelve las rutas que fueron (o serían)
// eliminadas.
func DeleteDuplicates(groups []DuplicateGroup, dryRun bool) ([]string, error) {
	return DeleteDuplicatesTo(os.Stdout, groups, dryRun)
}

// DeleteDuplicatesTo deletes only byte-identical copies and writes dry-run
// messages to output.
func DeleteDuplicatesTo(output io.Writer, groups []DuplicateGroup, dryRun bool) ([]string, error) {
	var removed []string
	for _, g := range groups {
		if len(g.Files) < 2 {
			continue
		}
		keeper := g.Files[0].Path
		keeperInfo, err := os.Stat(keeper)
		if err != nil {
			return removed, fmt.Errorf("statting keeper %s: %w", keeper, err)
		}
		for i := 1; i < len(g.Files); i++ {
			path := g.Files[i].Path
			candidateInfo, err := os.Stat(path)
			if err != nil {
				return removed, fmt.Errorf("statting candidate %s: %w", path, err)
			}
			if os.SameFile(keeperInfo, candidateInfo) {
				return removed, fmt.Errorf("refusing to delete %s: it refers to the kept file", path)
			}
			equal, err := filesEqual(keeper, path)
			if err != nil {
				return removed, fmt.Errorf("comparing %s with %s: %w", keeper, path, err)
			}
			if !equal {
				return removed, fmt.Errorf("refusing to delete %s: contents differ from %s", path, keeper)
			}
			if dryRun {
				if _, err := fmt.Fprintf(output, "[dry-run] would delete %s\n", path); err != nil {
					return removed, fmt.Errorf("writing dry-run output: %w", err)
				}
				removed = append(removed, path)
				continue
			}
			if err := os.Remove(path); err != nil {
				return removed, fmt.Errorf("deleting %s: %w", path, err)
			}
			removed = append(removed, path)
		}
	}
	return removed, nil
}

func filesEqual(pathA, pathB string) (bool, error) {
	fileA, err := os.Open(pathA)
	if err != nil {
		return false, err
	}
	defer fileA.Close()

	fileB, err := os.Open(pathB)
	if err != nil {
		return false, err
	}
	defer fileB.Close()

	bufferA := make([]byte, 32*1024)
	bufferB := make([]byte, len(bufferA))
	for {
		nA, errA := io.ReadFull(fileA, bufferA)
		nB, errB := io.ReadFull(fileB, bufferB)
		if nA != nB || !bytes.Equal(bufferA[:nA], bufferB[:nB]) {
			return false, nil
		}

		if errA == io.EOF && errB == io.EOF {
			return true, nil
		}
		if errA == io.ErrUnexpectedEOF && errB == io.ErrUnexpectedEOF {
			return true, nil
		}
		if errA != nil && errA != io.EOF && errA != io.ErrUnexpectedEOF {
			return false, errA
		}
		if errB != nil && errB != io.EOF && errB != io.ErrUnexpectedEOF {
			return false, errB
		}
		if (errA == io.EOF || errA == io.ErrUnexpectedEOF) != (errB == io.EOF || errB == io.ErrUnexpectedEOF) {
			return false, nil
		}
	}
}
