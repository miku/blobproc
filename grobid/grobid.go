package grobid

import (
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
)

type Grobid struct {
	Server string
}

// ProcessFulltext runs full analysis of a PDF against grobid. TODO: where to
// store the result.
func (g *Grobid) ProcessFulltext(filename string) error {

}

// File wraps a file to upload.
type File struct {
	Name     string
	File     io.ReadCloser
	MimeType string
}

// https://github.com/kermitt2/grobid_client_python/blob/1fa605ff13cdaf8218fdabbcd4f923d48c4868b9/grobid_client/grobid_client.py#L259-L266

func doPost(link string, params url.Values, headers http.Header, file File) {
	var (
		in, out = io.Pipe()
		w       = multipart.NewWriter(in)
		resp    *http.Response
		done    = make(chan error)
	)
	go func() {
		req, err := http.NewRequest("POST", url, out)
		if err != nil {
			done <- err
			return
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			done <- err
			return
		}
		done <- nil
	}()
	part, err := w.CreateFormFile("input", filepath.Base(file.Name))
	_, _ = io.Copy(part, file.File)
	w.Close()
	in.Clone()
	if err := <-done; err != nil {
		log.Fatal(err)
	}
	log.Println("upload done")
}
