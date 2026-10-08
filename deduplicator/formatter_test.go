package deduplicator

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestJSONPrintToEmitsPureJSONForEmptyReport(t *testing.T) {
	var output bytes.Buffer
	if err := JSONPrintTo(&output, nil); err != nil {
		t.Fatalf("JSONPrintTo: %v", err)
	}

	var report DuplicateReport
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if report.Groups == nil || len(report.Groups) != 0 {
		t.Fatalf("expected an empty groups array, got %#v", report.Groups)
	}
	if report.TotalGroups != 0 || report.TotalFiles != 0 || report.TotalWasted != 0 {
		t.Fatalf("expected empty totals, got %#v", report)
	}
}
