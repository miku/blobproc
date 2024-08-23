package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/miku/blobproc"
	"github.com/miku/blobproc/pdfextract"
	"github.com/miku/grobidclient"
)

var docs = `blobproc - process and persist PDF documents derivations

Emit JSON with locally extracted data:

  $ blobproc -f file.pdf | jq .

Flags
`

var (
	singleFile        = flag.String("f", "", "process a single file (local tools only), for testing")
	spoolDir          = flag.String("spool", path.Join(xdg.DataHome, "/blobproc/spool"), "")
	logFile           = flag.String("logfile", "", "structured log output file, stderr if empty")
	debug             = flag.Bool("debug", false, "more verbose output")
	timeout           = flag.Duration("T", 300*time.Second, "subprocess timeout")
	keepSpool         = flag.Bool("k", false, "keep files in spool after processing, mainly for debugging")
	showVersion       = flag.Bool("version", false, "show version")
	grobidHost        = flag.String("grobid-host", "http://localhost:8070", "grobid host, cf. https://is.gd/3wnssq") // TODO: add multiple servers
	grobidMaxFileSize = flag.Int64("grobid-max-filesize", 256*1024*1024, "max file size to send to grobid in bytes")
	s3Endpoint        = flag.String("s3-endpoint", "localhost:9000", "S3 endpoint")
	s3AccessKey       = flag.String("s3-access-key", "minioadmin", "S3 access key")
	s3SecretKey       = flag.String("s3-secret-key", "minioadmin", "S3 secret key")
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, docs)
		flag.PrintDefaults()
	}
	flag.Parse()
	switch {
	case *showVersion:
		fmt.Println(blobproc.Version)
	case *singleFile != "":
		// Run a single file through local commands only.
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		result := pdfextract.ProcessFile(ctx, *singleFile, &pdfextract.Options{
			Dim:       pdfextract.Dim{180, 300},
			ThumbType: "JPEG"},
		)
		if result.Err != nil {
			log.Fatal(result.Err)
		}
		if result.Status != "success" {
			log.Fatalf("process failed with: %v", result.Status)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			log.Fatal(err)
		}
	default:
		// By default, try to work through the whole spool dir.
		//
		// This whole block of code is reading files from disk, processing them
		// through various tools and services and persists the results in S3.
		// The spool directory is the queue and it gets cleanup up, once the
		// file has been processed, even if just partially.
		//
		// You should be able to just add files to the spool folder again to
		// process them and to overwrite previous results in S3.
		//
		// Various logging setups.
		var (
			logLevel = slog.LevelInfo
			h        slog.Handler
		)
		if *debug {
			logLevel = slog.LevelDebug
		}
		switch {
		case *logFile != "":
			f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				slog.Error("cannot open log", "err", err)
				os.Exit(1)
			}
			defer f.Close()
			h = slog.NewJSONHandler(f, &slog.HandlerOptions{Level: logLevel})
		default:
			h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
		}
		logger := slog.New(h)
		slog.SetDefault(logger)
		// Setup external services and data stores.
		grobid := grobidclient.New(*grobidHost)
		slog.Info("grobid client", "host", *grobidHost)
		s3opts := &blobproc.WrapS3Options{
			AccessKey:     strings.TrimSpace(*s3AccessKey),
			SecretKey:     strings.TrimSpace(*s3SecretKey),
			DefaultBucket: "sandcrawler",
			UseSSL:        false,
		}
		wrapS3, err := blobproc.NewWrapS3(*s3Endpoint, s3opts)
		if err != nil {
			slog.Error("cannot access S3", "err", err)
			log.Fatalf("cannot access S3: %v", err)
		}
		slog.Info("s3 wrapper", "endpoint", *s3Endpoint)
		// Walk the spool directory and process one file after another. Run
		// local tools and send PDF to grobid, persist all results into S3.
		//
		// Partial success is accepted. However, the original PDF file will be
		// removed from the spool folder. To reprocess, add the PDF to the spool folder again.
		started := time.Now()
		var stats struct {
			NumFiles   int // Total number of files seen in one pass.
			NumOK      int // All went fine.
			NumSkipped int // Skipped for any reason.
		}
		err = filepath.Walk(*spoolDir, func(path string, info fs.FileInfo, err error) error {
			stats.NumFiles++
			if err != nil {
				return err
			}
			if info.IsDir() {
				stats.NumSkipped++
				return nil
			}
			if info.Size() == 0 {
				stats.NumSkipped++
				slog.Warn("skipping empty file", "path", path)
				return nil
			}
			slog.Debug("processing", "path", path)
			defer func() {
				if !*keepSpool {
					if err := os.Remove(path); err != nil {
						slog.Warn("error removing file from spool", "err", err, "path", path)
					}
				} else {
					slog.Debug("keeping file in spool", "path", path)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), *timeout)
			defer cancel()
			// Fulltext and thumbail via local command line tools.
			result := pdfextract.ProcessFile(ctx, path, &pdfextract.Options{
				Dim:       pdfextract.Dim{180, 300},
				ThumbType: "JPEG",
			})
			switch {
			case result.Status != "success":
				slog.Warn("pdfextract failed", "status", result.Status, "err", result.Err)
			case len(result.SHA1Hex) != 40:
				slog.Warn("invalid sha1 in response", "sha1", result.SHA1Hex)
			case result.Status == "success":
				// If we have a thumbnail, save it.
				if result.HasPage0Thumbnail() {
					opts := blobproc.BlobRequestOptions{
						Bucket:  "thumbnail",
						Folder:  "pdf",
						Blob:    result.Page0Thumbnail,
						SHA1Hex: result.SHA1Hex,
						Ext:     "180px.jpg",
						Prefix:  "",
					}
					resp, err := wrapS3.PutBlob(ctx, &opts)
					if err != nil {
						slog.Error("s3 failed (thumbnail)", "err", err)
					} else {
						slog.Debug("s3 put ok", "bucket", resp.Bucket, "path", resp.ObjectPath)
					}
				}
				// If we have some text, save it.
				if len(result.Text) > 0 {
					opts := blobproc.BlobRequestOptions{
						Bucket:  "sandcrawler",
						Folder:  "text",
						Blob:    []byte(result.Text),
						SHA1Hex: result.SHA1Hex,
						Ext:     "txt",
						Prefix:  "",
					}
					resp, err := wrapS3.PutBlob(ctx, &opts)
					if err != nil {
						slog.Error("s3 failed (text)", "err", err)
					} else {
						slog.Debug("s3 put ok", "bucket", resp.Bucket, "path", resp.ObjectPath)
					}
				}
			}
			if info.Size() > *grobidMaxFileSize {
				slog.Warn("skipping too large file", "path", path, "size", info.Size())
				return nil
			}
			// Structured metadata from PDF via grobid.
			gres, err := grobid.ProcessPDFContext(ctx, path, "processFulltextDocument", &grobidclient.Options{
				GenerateIDs:            true,
				ConsolidateHeader:      true,
				ConsolidateCitations:   false, // "too expensive for now"
				IncludeRawCitations:    true,
				IncluseRawAffiliations: true,
				TEICoordinates:         []string{"ref", "figure", "persName", "formula", "biblStruct"},
				SegmentSentences:       true,
			})
			switch {
			case err != nil || gres.Err != nil:
				slog.Warn("grobid failed", "err", err)
				return nil
			default:
				opts := blobproc.BlobRequestOptions{
					Bucket:  "sandcrawler",
					Folder:  "grobid",
					Blob:    gres.Body,
					SHA1Hex: gres.SHA1Hex,
					Ext:     "tei.xml",
					Prefix:  "",
				}
				resp, err := wrapS3.PutBlob(ctx, &opts)
				if err != nil {
					slog.Error("s3 failed (text)", "err", err)
					return nil
				} else {
					slog.Debug("s3 put ok", "bucket", resp.Bucket, "path", resp.ObjectPath)
				}
			}
			stats.NumOK++
			return nil
		})
		if err != nil {
			slog.Error("walk failed", "err", err)
			os.Exit(1)
		}
		slog.Info("directory walk done", "t", time.Since(started), "ts", time.Since(started).String(), "total", stats.NumFiles, "ok", stats.NumOK, "skipped", stats.NumSkipped)
	}
}
