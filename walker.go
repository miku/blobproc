package blobproc

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/miku/blobproc/pdfextract"
	"github.com/miku/grobidclient"
)

// WalkStats are a poor mans metrics.
type WalkStats struct {
	Processed int
	OK        int
}

// SuccessRatio calculates the ration of successful to total processed files.
func (ws *WalkStats) SuccessRatio() float64 {
	if ws.Processed == 0 {
		return 1.0
	}
	return float64(ws.OK) / float64(ws.Processed)
}

// Payload is what we pass to workers. Since the worker needs file size
// information, we pass it along, as the expensive stat has already been
// performed.
type Payload struct {
	Path     string
	FileInfo fs.FileInfo
}

// WalkFast is a walker that runs postprocessing in parallel.
type WalkFast struct {
	Dir               string
	NumWorkers        int
	KeepSpool         bool
	GrobidMaxFileSize int64
	Timeout           time.Duration
	Grobid            *grobidclient.Grobid
	S3                *WrapS3
	mu                sync.Mutex
	stats             *WalkStats
}

// worker can process path from a queue in a thread. If the worker context is
// cancelled, it will wrap up the last processing step and then tear down.
func (w *WalkFast) worker(wctx context.Context, workerName string, queue chan Payload, wg *sync.WaitGroup) {
	defer wg.Done()
	logger := slog.With(
		slog.String("worker", workerName),
	)
	for payload := range queue {
		select {
		case <-wctx.Done():
			break
		default:
			wrapper := func() {
				path := payload.Path
				logger.Debug("processing", "path", path)
				started := time.Now()
				w.mu.Lock()
				w.stats.Processed++
				w.mu.Unlock()
				defer func() {
					if !w.KeepSpool {
						if _, err := os.Stat(path); err == nil {
							if err := os.Remove(path); err != nil {
								logger.Warn("error removing file from spool", "err", err, "path", path)
							}
						}
					} else {
						logger.Debug("keeping file in spool", "path", path)
					}
				}()
				ctx, cancel := context.WithTimeout(context.Background(), w.Timeout)
				defer cancel()
				// Fulltext and thumbail via local command line tools
				// --------------------------------------------------
				result := pdfextract.ProcessFile(ctx, path, &pdfextract.Options{
					Dim:       pdfextract.Dim{180, 300},
					ThumbType: "JPEG",
				})
				switch {
				case result.Status != "success":
					logger.Warn("pdfextract failed", "status", result.Status, "err", result.Err)
				case len(result.SHA1Hex) != 40:
					logger.Warn("invalid sha1 in response", "sha1", result.SHA1Hex)
				case result.Status == "success":
					// If we have a thumbnail, save it.
					if result.HasPage0Thumbnail() {
						opts := BlobRequestOptions{
							Bucket:  "thumbnail",
							Folder:  "pdf",
							Blob:    result.Page0Thumbnail,
							SHA1Hex: result.SHA1Hex,
							Ext:     "180px.jpg",
							Prefix:  "",
						}
						resp, err := w.S3.PutBlob(ctx, &opts)
						if err != nil {
							logger.Error("s3 failed (thumbnail)", "err", err, "sha1", result.SHA1Hex)
						} else {
							logger.Debug("s3 put ok", "bucket", resp.Bucket, "path", resp.ObjectPath)
						}
					}
					// If we have some text, save it.
					if len(result.Text) > 0 {
						opts := BlobRequestOptions{
							Bucket:  "sandcrawler",
							Folder:  "text",
							Blob:    []byte(result.Text),
							SHA1Hex: result.SHA1Hex,
							Ext:     "txt",
							Prefix:  "",
						}
						resp, err := w.S3.PutBlob(ctx, &opts)
						if err != nil {
							logger.Error("s3 failed (text)", "err", err, "sha1", result.SHA1Hex)
						} else {
							logger.Debug("s3 put ok", "bucket", resp.Bucket, "path", resp.ObjectPath)
						}
					}
				}
				if payload.FileInfo.Size() > w.GrobidMaxFileSize {
					logger.Warn("skipping too large file", "path", path, "size", payload.FileInfo.Size())
					return
				}
				// Structured metadata from PDF via grobid
				// ---------------------------------------
				gres, err := w.Grobid.ProcessPDFContext(ctx, path, "processFulltextDocument", &grobidclient.Options{
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
					logger.Warn("grobid failed", "err", err)
				default:
					opts := BlobRequestOptions{
						Bucket:  "sandcrawler",
						Folder:  "grobid",
						Blob:    gres.Body,
						SHA1Hex: gres.SHA1Hex,
						Ext:     "tei.xml",
						Prefix:  "",
					}
					resp, err := w.S3.PutBlob(ctx, &opts)
					if err != nil {
						logger.Error("s3 failed (text)", "err", err)
					} else {
						logger.Debug("s3 put ok", "bucket", resp.Bucket, "path", resp.ObjectPath)
					}
				}
				logger.Debug("processing finished successfully", "path", path, "t", time.Since(started), "ts", time.Since(started).Seconds)
				w.mu.Lock()
				w.stats.OK++
				w.mu.Unlock()
			}
			wrapper() // for defer
		}
	}
	logger.Debug("worker shutdown ok")
}

// Run start processing files. Do some basic sanity check before setting up
// workers as we do not have a constructor function.
func (w *WalkFast) Run(ctx context.Context) error {
	if w.Grobid == nil {
		return fmt.Errorf("walker needs grobid setup")
	}
	if w.S3 == nil {
		return fmt.Errorf("walker needs S3")
	}
	w.stats = new(WalkStats)
	var queue = make(chan Payload)
	var wg sync.WaitGroup
	for i := 0; i < w.NumWorkers; i++ {
		wg.Add(1)
		name := fmt.Sprintf("worker-%02d", i)
		go w.worker(ctx, name, queue, &wg)
	}
	err := filepath.Walk(w.Dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Size() == 0 {
			slog.Warn("skipping empty file", "path", path)
			return nil
		}
		slog.Debug("walk status", "total", w.stats.Processed, "success", w.stats.SuccessRatio())
		select {
		case queue <- Payload{Path: path, FileInfo: info}:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
	close(queue)
	wg.Wait()
	return err
}
