// blobprocd takes binary blobs via HTTP POST and save them to disk.
package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
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
	debug       = flag.Bool("debug", false, "switch to log level DEBUG")
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println(blobproc.Version)
		os.Exit(0)
	}
	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}
	svc := &blobproc.WebSpoolService{
		Dir:        *spoolDir,
		ListenAddr: *listenAddr,
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
