package blobproc

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const tempFilePattern = "blobprocd-*"

var errShortName = errors.New("short name")

// WebSpoolService saves web payload to a configured directory. TODO: add limit
// in size (e.g. 80% of disk or absolute value)
type WebSpoolService struct {
	Dir        string
	ListenAddr string
	// TODO: add a (optional) reference to a store for url content hashes; it
	// would be good to keep it optional (so one may just copy files into the
	// spool folder), and maybe to provide a simple interface that can be
	// easily fulfilled by different backend; it would be good to keep it
	// optional (so one may just copy files into the spool folder), and maybe
	// to provide a simple interface that can be easily fulfilled by different
	// backend.
	URLMap *URLMap
}

// spoolListEntry collects basic information about a spooled file.
type spoolListEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"t"`
	URL     string `json:"url"`
}

// shardedPath takes a filename (without path) and returns the full path
// including shards. If create is true, also create subdirectories, if
// necessary.
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

// shardedPathExists returns true, if the sharded path for a given filename exists.
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

// shardedPathToIdentifier return the SHA1, given a sharded path.
func shardedPathToIdentifier(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return ""
	}
	n := len(parts)
	return parts[n-3] + parts[n-2] + parts[n-1]
}

// SpoolListHandler returns a single, long jsonlines response with information
// about all files in the spool directory.
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
			URL:     fmt.Sprintf("http://%v/spool/%v", svc.ListenAddr, id),
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

// SpoolStatusHandler returns HTTP 200, if a given file is in the spool
// directory and HTTP 404, if the file is not in the spool directory.
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

// BlobHandler receives binary blobs and saves them on disk. This handler
// returns as soon as the file has been written into the spool directory of the
// service, using a sharded SHA1 as path.
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
		spoolURL = fmt.Sprintf("http://%v/spool/%v", svc.ListenAddr, digest)
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
	// If we use heritrix, we can capture the originating URL and log it as
	// well. TODO: get rid of this exception.
	curi := r.Header.Get("X-Heritrix-CURI")
	if curi != "" {
		slog.Debug("spooled file", "file", dst, "url", spoolURL, "t", time.Since(started), "curi", curi)
		// If we have a URLMap configured, try to record the url, sha1 pair.
		if svc.URLMap != nil {
			err := svc.URLMap.Insert(curi, digest)
			if err != nil {
				slog.Warn("could not update urlmap", "err", err, "url", curi, "sha1", digest)
			}
		}
	} else {
		slog.Debug("spooled file", "file", dst, "url", spoolURL, "t", time.Since(started))
	}
	w.Header().Add("Location", spoolURL)
	w.WriteHeader(http.StatusAccepted)
}
