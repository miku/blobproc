package main

import (
	"flag"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/miku/blobrun/pidfile"
)

var (
	spoolDir                              = flag.String("spool", path.Join(xdg.DataHome, "/webspool/spool"), "")
	pidFile                               = flag.String("pidfile", path.Join(xdg.RuntimeDir, "webspool.pid"), "pidfile")
	grobidHost                            = flag.String("grobid", "http://localhost:8070", "grobid host, cf. https://is.gd/3wnssq")
	grobidProcessFulltextDocumentEndpoint = flag.String("grobid-process-fulltext-document-endpoint", "/api/processFulltextDocument", "fulltext document parsing endpoint")
	s3                                    = flag.String("s3", "", "S3 endpoint") // TODO: access key in env
	s3GrobidBucket                        = flag.String("s3-grobid-bucket", "", "s3 grobid fulltext bucket")
	s3TextBucket                          = flag.String("s3-text-bucket", "", "s3 fulltext bucket")
	s3ThumbnailBucket                     = flag.String("s3-thumbnail-bucket", "", "s3 thumbnail bucket")
)

type SpoolRunner struct {
	SpoolDir     string
	GrobidRunner *GrobidRunner
}

type GrobidRunner struct {
	Server    string
	BatchSize int
	mu        sync.Mutex
	batch     []string
}

func (r *GrobidRunner) processFulltextDocumentURL() string {
	return strings.TrimRight(r.Server, "/") + *grobidProcessFulltextDocumentEndpoint
}

func (r *GrobidRunner) Add(filename string) {
	r.mu.Lock()
	r.batch = append(r.batch, filename)
	r.mu.Unlock()
}

func (r *GrobidRunner) RunBatch() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: use goroutines to process files in parallel
	for _, filename := range r.batch {
		params := url.Values{
			"generateIds":            []string{"1"},
			"consolidateHeader":      []string{"1"},
			"consolidateCitations":   []string{"1"},
			"includeRawCitations":    []string{"1"},
			"includeRawAffiliations": []string{"1"},
			"teiCoordinates":         []string{"1"},
			"segmentSentences":       []string{"1"},
		}
		http.NewRequest("POST", r.processFulltextDocumentURL(), params)
	}

}

func (sr *SpoolRunner) RunGrobid(filename string) error {
	return nil
}
func (sr *SpoolRunner) RunPdfToText(filename string) error    { return nil }
func (sr *SpoolRunner) RunPdfThumbnail(filename string) error { return nil }

func main() {
	flag.Parse()
	if err := pidfile.Write(*pidFile, os.Getpid()); err != nil {
		slog.Error("aborting", "err", err)
		os.Exit(1)
	}
	runner := &SpoolRunner{
		SpoolDir: *spoolDir,
	}
	err := filepath.Walk(*spoolDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		slog.Info("processing", "path", path)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
