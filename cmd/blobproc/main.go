package main

import (
	"flag"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/miku/blobrun"
	"github.com/miku/blobrun/pidfile"
	"github.com/miku/grobidclient"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	spoolDir          = flag.String("spool", path.Join(xdg.DataHome, "/webspool/spool"), "")
	pidFile           = flag.String("pidfile", path.Join(xdg.RuntimeDir, "webspool.pid"), "pidfile")
	grobidHost        = flag.String("grobid", "http://localhost:8070", "grobid host, cf. https://is.gd/3wnssq")
	consolidateMode   = flag.Bool("consolidate-mode", false, "consolidate mode")
	maxGrobidFilesize = flag.Int64("max-grobid-filesize", 256*1024*1024, "max file size to send to grobid in bytes")
	s3                = flag.String("s3", "", "S3 endpoint") // TODO: access key in env
	s3AccessKey       = flag.String("s3-access-key", "", "S3 access key")
	s3SecretKey       = flag.String("s3-secret-key", "", "S3 secret key")
	s3GrobidBucket    = flag.String("s3-grobid-bucket", "", "s3 grobid fulltext bucket")
	s3TextBucket      = flag.String("s3-text-bucket", "", "s3 fulltext bucket")
	s3ThumbnailBucket = flag.String("s3-thumbnail-bucket", "", "s3 thumbnail bucket")
)

func main() {
	flag.Parse()
	if err := pidfile.Write(*pidFile, os.Getpid()); err != nil {
		slog.Error("exiting", "err", err)
		os.Exit(1)
	}
	grobid := grobidclient.New(*grobidHost)
	s3Client, err := minio.New(*s3, &minio.Options{
		Creds:  credentials.NewStaticV4(*s3AccessKey, *s3SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("cannot access S3: %v", err)
	}
	runner := &blobrun.Runner{
		SpoolDir:          *spoolDir,
		Grobid:            grobid,
		MaxGrobidFileSize: *maxGrobidFilesize,
		ConsolidateMode:   *consolidateMode,
		S3Client:          s3Client,
	}
	err = filepath.Walk(*spoolDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		slog.Info("processing", "path", path)

		if err := runner.RunGrobid(path); err != nil {
			slog.Error("grobid failed", "err", err)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}