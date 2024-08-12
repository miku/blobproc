package blobproc

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// WrapS3 slightly wraps I/O around our S3 store with convenience methods.
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

// NewWrapS3 creates a new, slim wrapper around S3.
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

// BlobRequestOptions wraps the options to put a blob into storage.
type BlobRequestOptions struct {
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
// prefix. Panic if sha1hex is not a length 40 string.
func blobPath(folder, sha1hex, ext, prefix string) string {
	if len(ext) > 0 && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s%s/%s/%s/%s%s",
		prefix, folder, sha1hex[0:2], sha1hex[2:4], sha1hex, ext)
}

// putBlob takes a data to be put into S3 and saves it.
func (wrap *WrapS3) putBlob(req *BlobRequestOptions) (*PutBlobResponse, error) {
	if req.SHA1Hex == "" {
		h := sha1.New()
		_, err := io.Copy(h, bytes.NewReader(req.Blob))
		if err != nil {
			return nil, err
		}
		req.SHA1Hex = fmt.Sprintf("%x", h.Sum(nil))
	}
	if len(req.SHA1Hex) != 40 {
		return nil, ErrInvalidHash
	}
	objPath := blobPath(req.Folder, req.SHA1Hex, req.Ext, req.Prefix)
	if req.Bucket == "" {
		req.Bucket = DefaultBucket
	}
	ok, err := wrap.Client.BucketExists(context.Background(), req.Bucket)
	if err != nil {
		return nil, err
	}
	if !ok {
		opts := minio.MakeBucketOptions{}
		if err := wrap.Client.MakeBucket(context.TODO(), req.Bucket, opts); err != nil {
			return nil, err
		}
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
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	info, err := wrap.Client.PutObject(context.TODO(), req.Bucket, objPath,
		bytes.NewReader(req.Blob), int64(len(req.Blob)), opts)
	if err != nil {
		return nil, err
	}
	if info.Bucket != req.Bucket {
		return nil, fmt.Errorf("[put] bucket mismatch: %v", info.Bucket)
	}
	if info.Key != objPath {
		return nil, fmt.Errorf("[put] key mismatch: %v", info.Key)
	}
	return &PutBlobResponse{
		Bucket:     info.Bucket,
		ObjectPath: info.Key,
	}, nil
}

// getBlob returns the object bytes given a blob request.
func (wrap *WrapS3) getBlob(req *BlobRequestOptions) ([]byte, error) {
	objPath := blobPath(req.Folder, req.SHA1Hex, req.Ext, req.Prefix)
	if req.Bucket == "" {
		req.Bucket = DefaultBucket
	}
	object, err := wrap.Client.GetObject(context.TODO(), req.Bucket, objPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return io.ReadAll(object)
}
