package pdfextract

import (
	_ "embed"
	"encoding/json"
	"os"
	"reflect"
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
			filename: "../testdata/pdf/1906.02444.pdf",
			dim:      Dim{180, 300},
			status:   "success",
			snapshot: "../testdata/extract/1906.02444.json",
		},
		{
			filename: "../testdata/pdf/1906.11632.pdf",
			dim:      Dim{180, 300},
			status:   "success",
			snapshot: "../testdata/extract/1906.11632.json",
		},
		{
			filename: "../testdata/misc/wordle.py",
			dim:      Dim{180, 300},
			status:   "not-pdf",
		},
	}
	for _, c := range cases {
		result := ProcessFile(c.filename, &Options{
			Dim:       c.dim,
			ThumbType: "jpg",
		})
		if result.Status != c.status {
			t.Fatalf("got %v, want %v", result.Status, c.status)
		}
		if result.Status != "success" {
			continue
		}
		var want Result
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
		for _, v := range []*Result{
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

//go:embed testdata/pdf/1906.02444.pdf
var testdataPdf1 []byte

func TestGenerateFileInfo(t *testing.T) {
	var cases = []struct {
		data   []byte
		result FileInfo
	}{
		{
			data: []byte{},
			result: FileInfo{
				Size:      0,
				MD5Hex:    "d41d8cd98f00b204e9800998ecf8427e",
				SHA1Hex:   "da39a3ee5e6b4b0d3255bfef95601890afd80709",
				SHA256Hex: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				Mimetype:  "text/plain",
			},
		},
		{
			data: testdataPdf1,
			result: FileInfo{
				Size:      633850,
				MD5Hex:    "e04a100bc6a02efbf791566d6cb62bc9",
				SHA1Hex:   "4e6ca8dfc787a8b33e92773df3674fadf4d4cdb6",
				SHA256Hex: "31d0504caf44007be46d5aa64640dc2c1054aa7f4f404851f3a40c06d4ed7008",
				Mimetype:  "application/pdf",
			},
		},
	}
	for _, c := range cases {
		var fi FileInfo
		fi.FromBytes(c.data)
		if !reflect.DeepEqual(fi, c.result) {
			t.Fatalf("got %v, want %v", fi, c.result)
		}
	}
}

func BenchmarkPdfExtract(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = ProcessFile("testdata/pdf/1906.02444.pdf", &Options{
			Dim:       Dim{180, 300},
			ThumbType: "na",
		})
	}
}
