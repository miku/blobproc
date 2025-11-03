// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	warc "github.com/internetarchive/gowarc"
)

var (
	fromWarcFile = flag.String("W", "", "start with a WARC file")
	// TODO: CDX, item, collection
)

func main() {
	flag.Parse()
	switch {
	case *fromWarcFile != "":
		f, err := os.Open(*fromWarcFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		wr, err := warc.NewReader(f)
		if err != nil {
			log.Fatal(err)
		}
		for {
			record, err := wr.ReadRecord()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			uri := record.Header.Get("WARC-Target-URI")
			if len(uri) == 0 {
				continue
			}
			// fmt.Println(uri)
			warcContentType := record.Header.Get("Content-Type")
			if warcContentType != "application/http; msgtype=response" {
				continue
			}

			// Get the content length from WARC header
			contentLengthStr := record.Header.Get("Content-Length")
			contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
			if err != nil {
				log.Printf("skipping record with invalid content-length: %v", err)
				continue
			}

			// Limit the reader to the WARC content length
			r := io.LimitReader(record.Content, contentLength)

			resp, err := http.ReadResponse(bufio.NewReader(r), nil)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()
			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/pdf" {
				continue
			}
			fmt.Println(uri)
			fmt.Println(record.Offset, record.Size)
			f, err := os.CreateTemp("", "blobproc-pdf-*")
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			_, err = io.Copy(f, resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(f.Name())
		}
	default:
		log.Println("blobfetch")
	}
}
