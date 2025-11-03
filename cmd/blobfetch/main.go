// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/miku/blobproc/warcutil"
)

var (
	fromWarcFile = flag.String("W", "", "start with a WARC file")
	outputDir    = flag.String("o", "", "output directory")
	postURL      = flag.String("u", "", "POST extracted content to URL")
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
		var processor warcutil.Processor
		switch {
		case *outputDir != "":
			processor = &warcutil.DirProcessor{
				Dir:       *outputDir,
				Prefix:    "blobfetch-",
				Extension: ".pdf",
			}
		case *postURL != "":
		default:
			processor = warcutil.DebugProcessor
		}
		extractor := warcutil.Extractor{
			Filters: []warcutil.ResponseFilter{
				warcutil.PDFResponseFilter,
			},
			Processors: []warcutil.Processor{processor},
		}
		if err := extractor.Extract(f); err != nil {
			log.Fatal(err)
		}
	default:
		log.Println("blobfetch")
	}
}
