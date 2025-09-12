// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/miku/blobproc/fetchutils"
)

var (
	fromWarcFile   = flag.String("W", "", "start with a WARC file")
	fromCdxFile    = flag.String("X", "", "use a cdx file to discover pdfs")
	fromItem       = flag.String("I", "", "start from an item identifier")
	fromCollection = flag.String("C", "", "start from a collection identifier")
)

func main() {
	flag.Parse()
	switch {
	case *fromWarcFile != "":
		// wf := &fetchutils.WarcFetch{Location: *fromWarcFile}
		// if err := wf.Run(); err != nil {
		// 	log.Fatal(err)
		// }
		dir, err := os.MkdirTemp("", "blobfetch-warc-pdf-finder-*")
		if err != nil {
			log.Fatal(err)
		}
		if err := fetchutils.ProcessWARCForPDFs(*fromWarcFile, dir, true); err != nil {
			log.Fatal(err)
		}
		log.Println(dir)
	case *fromCdxFile != "":
	case *fromItem != "":
	case *fromCollection != "":
	}
}
