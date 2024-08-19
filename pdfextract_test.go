package blobproc

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPdfExtract(t *testing.T) {
	result := ProcessPDFFile("testdata/pdf/1906.02444.pdf", Dim{180, 300}, "na")
	var want PDFExtractResult
	var snapshot = "testdata/extract/1906.02444.json"
	b, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatalf("cannot read snapshot: %v", snapshot)
	}
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatalf("snapshot broken: %v", err)
	}
	// PDFCPU fields that change on every run.
	result.Metadata.PDFCPU.Header.Creation = ""
	result.Metadata.PDFCPU.Infos[0].Source = ""
	want.Metadata.PDFCPU.Header.Creation = ""
	want.Metadata.PDFCPU.Infos[0].Source = ""
	// Compare fixed fields now.
	if !cmp.Equal(result, &want) {
		t.Fatalf("diff: %v", cmp.Diff(result, &want))
	}
}

func BenchmarkPdfExtract(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = ProcessPDFFile("testdata/pdf/1906.02444.pdf", Dim{180, 300}, "na")
	}
}
