package blobproc

import (
	_ "embed"
	"reflect"
	"testing"
)

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
				Size:      0,
				MD5Hex:    "e04a100bc6a02efbf791566d6cb62bc9",
				SHA1Hex:   "4e6ca8dfc787a8b33e92773df3674fadf4d4cdb6",
				SHA256Hex: "31d0504caf44007be46d5aa64640dc2c1054aa7f4f404851f3a40c06d4ed7008",
				Mimetype:  "application/pdf",
			},
		},
	}
	for _, c := range cases {
		fi := GenerateFileInfo(c.data)
		if !reflect.DeepEqual(fi, c.result) {
			t.Fatalf("got %v, want %v", fi, c.result)
		}
	}
}
