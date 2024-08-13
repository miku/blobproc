package blobproc

import (
	"errors"
	"fmt"
	"os"

	"github.com/miku/grobidclient"
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
	S3Wrapper         *WrapS3              // Wraps access to S3/seaweedfs.
}

// ProcessFulltextResult is a wrapped grobid response. TODO: we may just use
// the GrobidResult directly.
type ProcessFulltextResult struct {
	StatusCode int
	Status     string
	Error      error
	TEIXML     string
	SHA1       string // SHA1 of the originating PDF, not the TEIXML
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
			StatusCode: -1,
			Error:      err,
			SHA1:       "",
		}, err
	}
	switch {
	case result.StatusCode == 200 && len(result.Body) > 12_000_000:
		err := fmt.Errorf("response XML too large: %d", len(result.Body))
		return &ProcessFulltextResult{
			Status: "error",
			Error:  err,
			SHA1:   result.SHA1,
		}, err
	case result.StatusCode == 200:
		return &ProcessFulltextResult{
			Status:     "success",
			StatusCode: result.StatusCode,
			TEIXML:     string(result.Body),
			Error:      nil,
			SHA1:       result.SHA1,
		}, nil
	default:
		return &ProcessFulltextResult{
			Status:     "error",
			StatusCode: result.StatusCode,
			Error:      fmt.Errorf("body: %v", string(result.Body)),
		}, nil
	}
}

// RunGrobid and persist, returns the sha1 of the filename and any error.
func (sr *Runner) RunGrobid(filename string) (string, error) {
	result, err := sr.processFulltext(filename)
	if err != nil {
		return "", err
	}
	opts := BlobRequestOptions{
		SHA1Hex: result.SHA1,
		Folder:  "grobid",
		Ext:     ".tei.xml",
		Bucket:  "sandcrawler",
	}
	_, err = sr.S3Wrapper.putBlob(&opts)
	return result.SHA1, err
}

func (sr *Runner) RunPdfToText(filename string) error { return nil }

func (sr *Runner) RunPdfThumbnail(filename string) error { return nil }
