package blobproc

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"strings"
)

// BlobS3 slightly wraps I/O around our S3 store.
type BlobS3 struct {
	HostURL       string
	AccessKey     string
	SecretKey     string
	DefaultBucket string
}

// PutBlobRequest wraps the options to put a blob into storage.
type PutBlobRequest struct {
	Folder  string
	Blob    []byte
	SHA1Hex string
	Ext     string
	Prefix  string
	Bucket  string
}

// PutBlobResponse wraps a blob put request response.
type PutBlobResponse struct {
	Bucket     string
	ObjectPath string
}

// blobPath panics, if the sha1hex is not a 40 byte hex digest.
func (b *BlobS3) blobPath(folder, sha1hex, ext, prefix string) string {
	if len(sha1hex) != 40 {
		panic(fmt.Sprintf("invalid sha1hex, want 40 bytes, got %v", len(sha1hex)))
	}
	if len(ext) > 0 && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s%s/%s/%s/%s%s",
		prefix, folder, sha1hex[0:2], sha1hex[2:4], sha1hex, ext)
}

func (b *BlobS3) putBlob(req *PutBlobRequest) (*PutBlobResponse, error) {
	if len(req.SHA1Hex) != 40 {
		return nil, ErrInvalidHash
	}
	if req.SHA1Hex == "" {
		h := sha1.New()
		_, err := io.Copy(h, bytes.NewReader(req.Blob))
		if err != nil {
			return nil, err
		}
		req.SHA1Hex = fmt.Sprintf("%x", h.Sum(nil))
	}
	objPath := b.blobPath(req.Folder, req.SHA1Hex, req.Ext, req.Prefix)
	log.Printf("TODO: objPath: %v", objPath)
	if req.Bucket == "" {
		req.Bucket = DefaultBucket
	}
	contentType := "application/octet-stream"
	if strings.HasSuffix(req.Ext, ".xml") {
		// TODO: contentType = "application/xml"
	}
	if strings.HasSuffix(req.Ext, ".png") {
	}
	if strings.HasSuffix(req.Ext, ".jpg") || strings.HasSuffix(req.Ext, ".jpeg") {
	}
	if strings.HasSuffix(req.Ext, ".txt") {
	}
	// TODO: minio put object

	return nil, nil
}

func (b *BlobS3) getBlob(req *PutBlobRequest) ([]byte, error) {
	return nil, nil
}
