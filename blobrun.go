package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// RunnerService calls a few external tools on the received payload.
type RunnerService struct {
	CacheDir string
}

func (svc *RunnerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tmpf, err := os.CreateTemp("", "blobrun-*")
	if err != nil {
		log.Printf("failed to create temporary file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	n, err := io.Copy(tmpf, r.Body)
	if err != nil {
		log.Printf("failed to drain response body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("received and stored file: %db", n)
	defer os.Remove(tmpf.Name())
}

func main() {
	svc := &RunnerService{}
	r := mux.NewRouter()
	r.HandleFunc("/p/1", svc.ServeHTTP)
	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("starting server at: %v", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
