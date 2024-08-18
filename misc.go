package blobproc

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"

	"github.com/gabriel-vasile/mimetype"
)

var ErrNoData = errors.New("no data")

// FileInfo groups checksum and size for a file. The checksums are all lowercase hex digests.
type FileInfo struct {
	Size      int64
	SHA1Hex   string
	SHA256Hex string
	MD5Hex    string
	Mimetype  string
}

// FromBytes creates a FileInfo object from bytes.
func (fi *FileInfo) FromBytes(p []byte) {
	var hasher = []hash.Hash{md5.New(), sha1.New(), sha256.New()}
	for _, h := range hasher {
		_, _ = h.Write(p)
	}
	*fi = FileInfo{
		Size:      int64(len(p)),
		MD5Hex:    hex.EncodeToString(hasher[0].Sum(nil)),
		SHA1Hex:   hex.EncodeToString(hasher[1].Sum(nil)),
		SHA256Hex: hex.EncodeToString(hasher[2].Sum(nil)),
		Mimetype:  mimetype.Detect(p).String(),
	}
}

// FromReader creates file info fields from metadata.
func (fi *FileInfo) FromReader(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	fi.FromBytes(b)
	return nil
}

// FromFile creates a FileInfo object from a path.
func (fi *FileInfo) FromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return fi.FromReader(f)
}
