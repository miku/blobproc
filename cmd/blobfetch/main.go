// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"flag"
	"log"

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
		wf := &fetchutils.WarcFetch{Location: *fromWarcFile}
		if err := wf.Run(); err != nil {
			log.Fatal(err)
		}
	case *fromCdxFile != "":
	case *fromItem != "":
	case *fromCollection != "":
	}
}
