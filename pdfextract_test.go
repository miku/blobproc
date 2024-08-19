package blobproc

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPdfExtract(t *testing.T) {
	var cases = []struct {
		filename string
		dim      Dim
		status   string
		snapshot string
	}{
		{
			filename: "testdata/pdf/1906.02444.pdf",
			dim:      Dim{180, 300},
			status:   "success",
			snapshot: "testdata/extract/1906.02444.json",
		},
		{
			filename: "testdata/pdf/1906.11632.pdf",
			dim:      Dim{180, 300},
			status:   "success",
			snapshot: "testdata/extract/1906.11632.json",
		},
		{
			filename: "testdata/misc/wordle.py",
			dim:      Dim{180, 300},
			status:   "not-pdf",
		},
	}
	for _, c := range cases {
		result := ProcessPDFFile(c.filename, &ProcessPDFOptions{
			Dim:       c.dim,
			ThumbType: "jpg",
		})
		if result.Status != c.status {
			t.Fatalf("got %v, want %v", result.Status, c.status)
		}
		if result.Status != "success" {
			continue
		}
		var want PDFExtractResult
		if _, err := os.Stat(c.snapshot); os.IsNotExist(err) {
			f, err := os.CreateTemp("", "blobproc-pdf-snapshot-*.json")
			if err != nil {
				t.Fatalf("could not create snapshot: %v", err)
			}
			defer f.Close()
			if err := json.NewEncoder(f).Encode(result); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			if err := os.Rename(f.Name(), c.snapshot); err != nil {
				t.Fatalf("rename: %v", err)
			}
			t.Logf("created new snapshot: %v", c.snapshot)
		}
		b, err := os.ReadFile(c.snapshot)
		if err != nil {
			t.Fatalf("cannot read snapshot: %v", c.snapshot)
		}
		if err := json.Unmarshal(b, &want); err != nil {
			t.Fatalf("snapshot broken: %v", err)
		}
		// PDFCPU fields that change on every run.
		for _, v := range []*PDFExtractResult{
			result,
			&want,
		} {
			v.Metadata.PDFCPU.Header.Creation = ""
			v.Metadata.PDFCPU.Infos[0].Source = ""
		}
		// Remaining fields should be fixed now.
		if !cmp.Equal(result, &want) {
			// If we fail, we write the result JSON to a tempfile for later
			// inspection or snapshot creation.
			t.Fatalf("diff: %v", cmp.Diff(result, &want))
		}
	}
}

func BenchmarkPdfExtract(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = ProcessPDFFile("testdata/pdf/1906.02444.pdf", &ProcessPDFOptions{
			Dim:       Dim{180, 300},
			ThumbType: "na",
		})
	}
}
