package blobproc

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"

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

func GenerateFileInfo(p []byte) FileInfo {
	var hasher = []hash.Hash{md5.New(), sha1.New(), sha256.New()}
	for _, h := range hasher {
		_, _ = h.Write(p)
	}
	return FileInfo{
		Size:      int64(len(p)),
		MD5Hex:    fmt.Sprintf("%x", hasher[0].Sum(nil)),
		SHA1Hex:   fmt.Sprintf("%x", hasher[1].Sum(nil)),
		SHA256Hex: fmt.Sprintf("%x", hasher[2].Sum(nil)),
		Mimetype:  mimetype.Detect(p).String(),
	}
}
