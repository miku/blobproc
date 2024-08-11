package blobproc

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// WrapS3 slightly wraps I/O around our S3 store.
type WrapS3 struct {
	Client *minio.Client
}

// WrapS3Options mostly contains pass through options for minio client.
type WrapS3Options struct {
	AccessKey     string
	SecretKey     string
	DefaultBucket string
	UseSSL        bool
}

func NewWrapS3(endpoint string, opts *WrapS3Options) (*WrapS3, error) {
	client, err := minio.New(endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
			Secure: opts.UseSSL,
		},
	)
	if err != nil {
		return nil, err
	}
	return &WrapS3{
		Client: client,
	}, nil
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

// blobPath returns the path for a given folder, content hash, extension and
// prefix.
func blobPath(folder, sha1hex, ext, prefix string) string {
	if len(ext) > 0 && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s%s/%s/%s/%s%s",
		prefix, folder, sha1hex[0:2], sha1hex[2:4], sha1hex, ext)
}

// putBlob takes a data to be put into S3 and saves it.
func (wrap *WrapS3) putBlob(req *PutBlobRequest) (*PutBlobResponse, error) {
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
	objPath := blobPath(req.Folder, req.SHA1Hex, req.Ext, req.Prefix)
	if req.Bucket == "" {
		req.Bucket = DefaultBucket
	}
	contentType := "application/octet-stream"
	if strings.HasSuffix(req.Ext, ".xml") {
		contentType = "application/xml"
	}
	if strings.HasSuffix(req.Ext, ".png") {
		contentType = "image/png"
	}
	if strings.HasSuffix(req.Ext, ".jpg") || strings.HasSuffix(req.Ext, ".jpeg") {
		contentType = "image/jpeg"
	}
	if strings.HasSuffix(req.Ext, ".txt") {
		contentType = "text/plain"
	}
	// TODO: minio put object
	log.Println(objPath, contentType)
	// wrap.Client.PutObject() // TODO: minio client
	return nil, nil
}

func (b *WrapS3) getBlob(req *PutBlobRequest) ([]byte, error) {
	return nil, nil
}
