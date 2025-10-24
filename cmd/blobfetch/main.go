// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

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
			fmt.Println(uri)
		}
	default:
		log.Println("blobfetch")
	}
}
