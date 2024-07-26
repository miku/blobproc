package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/adrg/xdg"
	"github.com/gorilla/mux"
)

var (
	spoolDir   = flag.String("spool", path.Join(xdg.DataHome, "/blobrun/spool"), "")
	listenAddr = flag.String("addr", "0.0.0.0:8000", "host port to listen on")
	timeout    = flag.Duration("T", 15*time.Second, "server timeout")

	banner = `{"id": "blobrun",
	"about": "Send your PDF payload to %s/p/1 - a 200 OK status only confirms
	receipt, not successful postprocessing, which may take more time."}`
)

type DeriveRunner struct {
	SpoolDir string
	// TODO: add storage locations
}

// Run runs all derivations and on success removes the file from the spool
// directory.
func (r *DeriveRunner) Run() error { return nil }

// RunnerService calls a few external tools on the received payload.
type RunnerService struct {
	SpoolDir string
}

func (svc *RunnerService) SpoolStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	digest := vars["id"]
	if len(digest) != 40 {
		log.Printf("invalid id: %v", digest)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var (
		shard   = digest[:2]
		dstDir  = path.Join(svc.SpoolDir, shard)
		dstPath = path.Join(dstDir, digest)
	)
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		// TODO: report age of file, etc.
		return
	}
}

// BlogHandler receives PDF blobs and saves them on disk.
func (svc *RunnerService) BlogHandler(w http.ResponseWriter, r *http.Request) {
	tmpf, err := os.CreateTemp("", "blobrun-*")
	if err != nil {
		log.Printf("failed to create temporary file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpf.Name())
	var (
		h  = sha1.New()
		mw = io.MultiWriter(h, tmpf)
	)
	n, err := io.Copy(mw, r.Body)
	if err != nil {
		log.Printf("failed to drain response body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tmpf.Close(); err != nil {
		log.Printf("failed to close temporary file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if n != r.ContentLength {
		log.Printf("content length mismatch, got %v, expected %v", n, r.ContentLength)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var (
		digest   = fmt.Sprintf("%x", h.Sum(nil))
		shard    = digest[:2]
		dstDir   = path.Join(svc.SpoolDir, shard)
		dstPath  = path.Join(dstDir, digest)
		spoolURL = fmt.Sprintf("http://%v/p/1/spool/%v", *listenAddr, digest)
	)
	if _, err := os.Stat(dstPath); err == nil {
		f, err := os.Open(dstPath)
		if err != nil {
			log.Printf("already uploaded, but file not readable: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			log.Printf("failed to stat file: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.ContentLength == fi.Size() {
			log.Printf("found existing file in spool dir, skipping, spool url: %v", spoolURL)
			w.Header().Add("Location", spoolURL)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		log.Printf("warning: found existing file, but size differ, overwriting")
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		log.Printf("failed to create directories: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := os.Rename(tmpf.Name(), dstPath); err != nil {
		log.Printf("failed to rename: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("spooled file to: %v, spool url: %v", dstPath, spoolURL)
	w.Header().Add("Location", spoolURL)
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	flag.Parse()
	svc := &RunnerService{
		SpoolDir: *spoolDir,
	}
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, banner+"\n", *listenAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	r.HandleFunc("/p/1", svc.BlogHandler)
	r.HandleFunc("/p/1/spool/{id}", svc.SpoolStatusHandler)
	srv := &http.Server{
		Handler:      r,
		Addr:         *listenAddr,
		WriteTimeout: *timeout,
		ReadTimeout:  *timeout,
	}
	log.Printf("starting server at: %v", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
