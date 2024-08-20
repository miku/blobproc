package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/miku/blobproc"
	"github.com/miku/blobproc/pdfextract"
	"github.com/miku/blobproc/pidfile"
	"github.com/miku/grobidclient"
)

var (
	singleFile = flag.String("f", "", "process a single file")

	spoolDir = flag.String("spool", path.Join(xdg.DataHome, "/blobproc/spool"), "")
	pidFile  = flag.String("pidfile", path.Join(xdg.RuntimeDir, "webspool.pid"), "pidfile")
	logFile  = flag.String("log", "", "structured log output file, stderr if empty")
	debug    = flag.Bool("debug", false, "more verbose output")
	timeout  = flag.Duration("T", 300*time.Second, "subprocess timeout")

	grobidHost        = flag.String("grobid", "http://localhost:8070", "grobid host, cf. https://is.gd/3wnssq") // TODO: add multiple servers
	consolidateMode   = flag.Bool("consolidate-mode", false, "consolidate mode")
	maxGrobidFilesize = flag.Int64("max-grobid-filesize", 256*1024*1024, "max file size to send to grobid in bytes")

	s3          = flag.String("s3", "", "S3 endpoint") // TODO: access key in env
	s3AccessKey = flag.String("s3-access-key", "minioadmin", "S3 access key")
	s3SecretKey = flag.String("s3-secret-key", "minioadmin", "S3 secret key")
)

func main() {
	flag.Parse()
	if *singleFile != "" {
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
			log.Fatal("process failed with: %v", result.Status)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if err := pidfile.Write(*pidFile, os.Getpid()); err != nil {
		slog.Error("exiting", "err", err, "pidfile", "*pidFile")
		os.Exit(1)
	}
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
	grobid := grobidclient.New(*grobidHost)
	slog.Info("initialize grobid client", "host", *grobidHost)
	s3wrapper, err := blobproc.NewWrapS3(*s3, &blobproc.WrapS3Options{
		AccessKey:     *s3AccessKey,
		SecretKey:     *s3SecretKey,
		DefaultBucket: "sandcrawler",
		UseSSL:        false,
	})
	if err != nil {
		slog.Error("cannot access S3", "err", err)
		os.Exit(1)
	}
	slog.Info("initialized s3 wrapper", "host", *s3)
	runner := &blobproc.Runner{
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
			slog.Error("grobid failed", "err", err, "path", path)
			return err
		}
		if err := runner.RunPdfToText(path); err != nil {
			slog.Error("pdftotext failed", "err", err, "path", path)
			return err
		}
		return nil
	})
	if err != nil {
		slog.Error("walk failed", "err", err)
		os.Exit(1)
	}
}
