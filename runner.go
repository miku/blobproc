package blobproc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/miku/grobidclient"
	"github.com/minio/minio-go/v7"
)

var (
	ErrFileTooLarge = errors.New("file too large")
	ErrInvalidHash  = errors.New("invalid hash")
)

var DefaultBucket = "default" // TODO: what is it?

// Runner run derivations of a file and also stores the results in S3.
type Runner struct {
	SpoolDir          string               // SpoolDir is a directory where we expect PDF files.
	Grobid            *grobidclient.Grobid // Grobid client wraps grobid service API access.
	MaxGrobidFileSize int64                // Do not send too large blobs to grobid.
	ConsolidateMode   bool                 // ConsolidateMode pass through argument to grobid.
	S3Client          *minio.Client        // S3Client wraps access to S3/seaweedfs.
}

// ProcessFulltextResult is a wrapped grobid response. TODO: we may just use
// the GrobidResult directly.
type ProcessFulltextResult struct {
	StatusCode int
	Status     string
	Error      error
	TEIXML     string
}

// processFulltext wrap grobid access and returns parsed document or some
// information about errors.
func (runner *Runner) processFulltext(filename string) (*ProcessFulltextResult, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	if fi.Size() > runner.MaxGrobidFileSize {
		return &ProcessFulltextResult{
			Status: "blob-too-large", // TODO: not sure we need that immediately
			Error:  ErrFileTooLarge,
		}, ErrFileTooLarge
	}
	opts := &grobidclient.Options{
		ConsolidateHeader:      runner.ConsolidateMode,
		ConsolidateCitations:   false, // "too expensive for now"
		IncludeRawCitations:    true,
		IncluseRawAffiliations: true,
		TEICoordinates:         []string{"ref", "figure", "persName", "formula", "biblStruct"},
		SegmentSentences:       true,
	}
	result, err := runner.Grobid.ProcessPDF(filename, "processFulltextDocument", opts)
	if err != nil {
		return &ProcessFulltextResult{
			Status:     "grobid-error",
			StatusCode: result.StatusCode,
			Error:      err,
		}, err
	}
	if result.StatusCode == 200 {
		if len(result.Body) > 12_000_000 {
			err := fmt.Errorf("response XML too large: %d", len(result.Body))
			return &ProcessFulltextResult{
				Status: "error",
				Error:  err,
			}, err
		}
		return &ProcessFulltextResult{
			Status:     "success",
			StatusCode: result.StatusCode,
			TEIXML:     string(result.Body),
			Error:      nil,
		}, nil
	}
	return &ProcessFulltextResult{
		Status:     "error",
		StatusCode: result.StatusCode,
		Error:      fmt.Errorf("body: %v", string(result.Body)),
	}, nil
}

func (sr *Runner) RunGrobid(filename string) error {
	_, err := sr.processFulltext(filename)
	if err != nil {
		return err
	}
	// Put result into bucket
	_, err = sr.S3Client.PutObject(
		context.TODO(),
		"my-bucketname",
		"my-objectname",
		bytes.NewReader(nil),
		0,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return err
	}
	return nil
}
func (sr *Runner) RunPdfToText(filename string) error { return nil }

func (sr *Runner) RunPdfThumbnail(filename string) error { return nil }