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
	"github.com/miku/blobproc"
	"github.com/miku/blobproc/pidfile"
	"github.com/miku/grobidclient"
)

var (
	spoolDir          = flag.String("spool", path.Join(xdg.DataHome, "/webspool/spool"), "")
	pidFile           = flag.String("pidfile", path.Join(xdg.RuntimeDir, "webspool.pid"), "pidfile")
	grobidHost        = flag.String("grobid", "http://localhost:8070", "grobid host, cf. https://is.gd/3wnssq")
	consolidateMode   = flag.Bool("consolidate-mode", false, "consolidate mode")
	maxGrobidFilesize = flag.Int64("max-grobid-filesize", 256*1024*1024, "max file size to send to grobid in bytes")
	s3                = flag.String("s3", "", "S3 endpoint") // TODO: access key in env
	s3AccessKey       = flag.String("s3-access-key", "minioadmin", "S3 access key")
	s3SecretKey       = flag.String("s3-secret-key", "minioadmin", "S3 secret key")
)

func main() {
	flag.Parse()
	if err := pidfile.Write(*pidFile, os.Getpid()); err != nil {
		slog.Error("exiting", "err", err)
		os.Exit(1)
	}
	grobid := grobidclient.New(*grobidHost)
	s3wrapper, err := blobproc.NewWrapS3(*s3, &blobproc.WrapS3Options{
		AccessKey:     *s3AccessKey,
		SecretKey:     *s3SecretKey,
		DefaultBucket: "sandcrawler",
		UseSSL:        false,
	})
	if err != nil {
		log.Fatalf("cannot access S3: %v", err)
	}
	runner := &blobproc.Runner{
		SpoolDir:          *spoolDir,
		Grobid:            grobid,
		MaxGrobidFileSize: *maxGrobidFilesize,
		ConsolidateMode:   *consolidateMode,
		S3Wrapper:         s3wrapper,
	}
	err = filepath.Walk(*spoolDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		slog.Info("processing", "path", path)
		if _, err := runner.RunGrobid(path); err != nil {
			slog.Error("grobid failed", "err", err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
