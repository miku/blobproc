// webspoold takes binary blobs via HTTP POST and save them to disk.
package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/gorilla/mux"
	"github.com/miku/blobproc"
)

var (
	spoolDir   = flag.String("spool", path.Join(xdg.DataHome, "/webspool/spool"), "")
	listenAddr = flag.String("addr", "0.0.0.0:8000", "host port to listen on")
	timeout    = flag.Duration("T", 15*time.Second, "server timeout")

	banner = `{"id": "webspool",
	"about": "Send your PDF payload to %s/spool - a 200 OK status only confirms
	receipt, not successful postprocessing, which may take more time."}`
	showVersion = flag.Bool("v", false, "show version")
)

var errShortName = errors.New("short name")

const tempFilePattern = "webspoold-*"

// WebSpoolService saves web payload to a configured directory. TODO: add limit
// in size (e.g. 80% of disk or absolute value)
type WebSpoolService struct {
	Dir string
}

// spoolListEntry collects basic information about a spooled file.
type spoolListEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"t"`
	URL     string `json:"url"`
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

func shardedPathToIdentifier(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return ""
	}
	n := len(parts)
	return parts[n-3] + parts[n-2] + parts[n-1]
}

func (svc *WebSpoolService) SpoolListHandler(w http.ResponseWriter, r *http.Request) {
	var (
		entry spoolListEntry
		enc   = json.NewEncoder(w)
	)
	err := filepath.Walk(svc.Dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		id := shardedPathToIdentifier(path)
		if len(id) == 0 {
			slog.Error("zero length id")
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("zero length id")
		}
		entry = spoolListEntry{
			Name:    id,
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
			URL:     fmt.Sprintf("http://%v/spool/%v", *listenAddr, id),
		}
		if err := enc.Encode(entry); err != nil {
			slog.Error("encoding error", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}
		return nil
	})
	if err != nil {
		slog.Error("failed to list files", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (svc *WebSpoolService) SpoolStatusHandler(w http.ResponseWriter, r *http.Request) {
	var (
		vars   = mux.Vars(r)
		digest = vars["id"]
	)
	if len(digest) != 40 {
		slog.Debug("invalid id", "id", digest)
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

// BlobHandler receives binary (PDF) blobs and saves them on disk.
func (svc *WebSpoolService) BlobHandler(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	tmpf, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		slog.Error("failed to create temporary file", "err", err)
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
		slog.Error("failed to drain response body", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tmpf.Close(); err != nil {
		slog.Error("failed to close temporary file", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if n != r.ContentLength {
		slog.Error("content length mismatch", "n", n, "length", r.ContentLength)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var (
		digest   = fmt.Sprintf("%x", h.Sum(nil))
		spoolURL = fmt.Sprintf("http://%v/spool/%v", *listenAddr, digest)
	)
	dst, err := svc.shardedPath(digest, true)
	if err != nil {
		slog.Error("could not determine sharded path", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ok, err := svc.shardedPathExists(digest)
	if err != nil {
		slog.Error("could not determine sharded path", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ok {
		f, err := os.Open(dst)
		if err != nil {
			slog.Error("already uploaded, but file not readable", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			slog.Error("failed to stat file", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.ContentLength == fi.Size() {
			slog.Debug("found existing file in spool dir, skipping", "url", spoolURL)
			w.Header().Add("Location", spoolURL)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		slog.Debug("warning: found existing file, but size differ, overwriting")
	}
	if err := os.Rename(tmpf.Name(), dst); err != nil {
		slog.Error("failed to rename", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	slog.Debug("spooled file", "file", dst, "url", spoolURL, "t", time.Since(started))
	w.Header().Add("Location", spoolURL)
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println(blobproc.Version)
		os.Exit(0)
	}
	slog.SetLogLoggerLevel(slog.LevelDebug)
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
	r.HandleFunc("/spool", svc.BlobHandler).Methods("POST")
	r.HandleFunc("/spool", svc.SpoolListHandler).Methods("GET")
	r.HandleFunc("/spool/{id}", svc.SpoolStatusHandler).Methods("GET")
	srv := &http.Server{
		Handler:      r,
		Addr:         *listenAddr,
		WriteTimeout: *timeout,
		ReadTimeout:  *timeout,
	}
	slog.Info("starting server at", "hostport", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
