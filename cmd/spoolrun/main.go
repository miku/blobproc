package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/miku/blobrun/pidfile"
	"github.com/miku/grobidclient"
	"github.com/minio/minio-go/pkg/credentials"
	"github.com/minio/minio-go/v7"
)

var (
	spoolDir          = flag.String("spool", path.Join(xdg.DataHome, "/webspool/spool"), "")
	pidFile           = flag.String("pidfile", path.Join(xdg.RuntimeDir, "webspool.pid"), "pidfile")
	grobidHost        = flag.String("grobid", "http://localhost:8070", "grobid host, cf. https://is.gd/3wnssq")
	s3                = flag.String("s3", "", "S3 endpoint") // TODO: access key in env
	s3AccessKey       = flag.String("s3-access-key", "", "S3 access key")
	s3SecretKey       = flag.String("s3-secret-key", "", "S3 secret key")
	s3GrobidBucket    = flag.String("s3-grobid-bucket", "", "s3 grobid fulltext bucket")
	s3TextBucket      = flag.String("s3-text-bucket", "", "s3 fulltext bucket")
	s3ThumbnailBucket = flag.String("s3-thumbnail-bucket", "", "s3 thumbnail bucket")
)

type Runner struct {
	SpoolDir string
	Grobid   *grobidclient.Grobid
	S3Client *minio.Client
}

func (sr *Runner) RunGrobid(filename string) error {
	return nil
}
func (sr *Runner) RunPdfToText(filename string) error    { return nil }
func (sr *Runner) RunPdfThumbnail(filename string) error { return nil }

func main() {
	flag.Parse()
	if err := pidfile.Write(*pidFile, os.Getpid()); err != nil {
		slog.Error("exiting", "err", err)
		os.Exit(1)
	}
	grobid := grobidclient.New(*grobidHost)
	s3Client, err := minio.New(*s3, &minio.Options{
		Creds:  credentials.NewStaticV4(*s3AccessKey, s3SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("cannot access S3: %v", err)
	}
	runner := &Runner{
		SpoolDir: *spoolDir,
		Grobid:   grobid,
		S3Client: s3Client,
	}
	err := filepath.Walk(*spoolDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		slog.Info("processing", "path", path)
		result, err := runner.Grobid.ProcessPDF(path, "processFulltextDocument", nil)
		if err != nil {
			slog.Warn("grobid pdf parsing failed", "err", err)
			return nil
		}
		if result.StatusCode != 200 {
			slog.Warn("grobid pdf parsing resulted in %d, skipping", "statuscode", result.StatusCode)
			return nil
		}
		// TODO: write successful result to S3 bucket
		info, err := runner.s3Client.PutObject(
			context.Background(),
			"my-bucketname",
			"my-objectname",
			object,
			objectStat.Size(),
			minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			log.Fatalln(err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
