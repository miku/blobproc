// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"flag"
	"io"
	"log"
	"os"

	warc "github.com/internetarchive/gowarc"
)

var (
	fromWarcFile = flag.String("W", "", "start with a WARC file")
	// fromCdxFile    = flag.String("X", "", "use a cdx file to discover pdfs")
	// fromItem       = flag.String("I", "", "start from an item identifier")
	// fromCollection = flag.String("C", "", "start from a collection identifier")
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
			log.Println(record.Header)
		}
	default:
		log.Println("blobfetch")
	}
}
