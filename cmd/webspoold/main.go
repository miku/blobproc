package main

import (
	"crypto/sha1"
	"errors"
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
	spoolDir   = flag.String("spool", path.Join(xdg.DataHome, "/webspool/spool"), "")
	listenAddr = flag.String("addr", "0.0.0.0:8000", "host port to listen on")
	timeout    = flag.Duration("T", 15*time.Second, "server timeout")

	banner = `{"id": "blobrun",
	"about": "Send your PDF payload to %s/spool - a 200 OK status only confirms
	receipt, not successful postprocessing, which may take more time."}`
)

var errShortName = errors.New("short name")

const tempFilePattern = "webspoold-*"

// WebSpoolService saves web payload to a configured directory.
type WebSpoolService struct {
	Dir string
}

// shardedPath takes a filename and returns the full path, including shards. If
// create is true, create subdirectories.
func (svc *WebSpoolService) shardedPath(filename string, create bool) (string, error) {
	if len(filename) < 8 {
		return "", errShortName
	}
	var (
		s0, s1 = filename[0:2], filename[2:4]
		dstDir = path.Join(svc.Dir, s0, s1)
	)
	if create {
		if _, err := os.Stat(dstDir); os.IsNotExist(err) {
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				return "", err
			}
		}
	}
	return path.Join(dstDir, filename[4:]), nil
}

func (svc *WebSpoolService) shardedPathExists(filename string) (bool, error) {
	dst, err := svc.shardedPath(filename, false)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(dst); err == nil {
		return true, nil
	}
	return false, nil
}

func (svc *WebSpoolService) SpoolStatusHandler(w http.ResponseWriter, r *http.Request) {
	var (
		vars   = mux.Vars(r)
		digest = vars["id"]
	)
	if len(digest) != 40 {
		log.Printf("invalid id: %v", digest)
		w.WriteHeader(http.StatusBadRequest)
	} else {
		ok, err := svc.shardedPathExists(digest)
		switch {
		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
		case ok:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// BlobHandler receives PDF blobs and saves them on disk.
func (svc *WebSpoolService) BlobHandler(w http.ResponseWriter, r *http.Request) {
	tmpf, err := os.CreateTemp("", tempFilePattern)
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
		spoolURL = fmt.Sprintf("http://%v/spool/%v", *listenAddr, digest)
	)
	dst, err := svc.shardedPath(digest, true)
	if err != nil {
		log.Printf("could not determine sharded path: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ok, err := svc.shardedPathExists(digest)
	if err != nil {
		log.Printf("could not determine sharded path: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ok {
		f, err := os.Open(dst)
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
	if err := os.Rename(tmpf.Name(), dst); err != nil {
		log.Printf("failed to rename: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("spooled file to: %v, spool url: %v", dst, spoolURL)
	w.Header().Add("Location", spoolURL)
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	flag.Parse()
	svc := &WebSpoolService{
		Dir: *spoolDir,
	}
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, banner+"\n", *listenAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	r.HandleFunc("/spool", svc.BlobHandler)
	r.HandleFunc("/spool/{id}", svc.SpoolStatusHandler)
	srv := &http.Server{
		Handler:      r,
		Addr:         *listenAddr,
		WriteTimeout: *timeout,
		ReadTimeout:  *timeout,
	}
	log.Printf("starting server at: %v", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
