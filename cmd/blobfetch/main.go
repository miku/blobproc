// blobfetch finds and fetches files from archive collections to be put into a
// spool folder for postprocessing. Scope of this tool is mostly PDF discovery
// and access.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/miku/blobproc/ia"
	"github.com/miku/blobproc/warcutil"
)

const (
	appName   = "blobproc"
	extracted = "extracted"
)

var (
	fromItem     = flag.String("I", "", "item name, e.g. 'HELLO-CRAWL-2020-10-20250920135023817-00102-00152-123'")
	fromWarcFile = flag.String("W", "", "start with a local WARC file")
	outputDir    = flag.String("o", "", "output directory, by default, use users cache dir")
	postURL      = flag.String("u", "", "POST extracted content to this URL")
	// TODO: CDX, item, collection
)

func main() {
	flag.Parse()
	switch {
	case *fromItem != "":
		// prepare cache dir
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			log.Fatal(err)
		}
		appCacheDir := path.Join(cacheDir, appName, extracted)
		if err := os.MkdirAll(appCacheDir, 0755); err != nil {
			log.Fatal(err)
		}
		// ex: https://archive.org/metadata/OPENALEX-CRAWL-2025-09-20251011130616382-07663-07716-wbgrp-crawl047
		link := fmt.Sprintf("https://archive.org/metadata/%s", *fromItem)
		log.Printf("fetching: %s", link)
		resp, err := http.Get(link)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatal("%v while fetching %s", resp.StatusCode, link)
		}
		var item ia.Item
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&item); err != nil {
			log.Fatal(err)
		}
		log.Printf("found %d files in %s", len(item.Files), *fromItem)
		for i, file := range item.Files {
			if !strings.HasSuffix(file.Name, ".warc.gz") {
				continue
			}
			// https://archive.org/download/OPENALEX-CRAWL-2025-09-20251011130616382-07663-07716-wbgrp-crawl047/OPENALEX-CRAWL-2025-09-20251011144946523-07666-2129926~wbgrp-crawl047.us.archive.org~8443.warc.gz
			fileURL := fmt.Sprintf("https://archive.org/download/%s/%s", *fromItem, file.Name)
			resp, err := http.Get(fileURL)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			// sample extractor that would only output the pdf url as it is found in the warc
			var debugProcessor = warcutil.FuncProcessor(func(e *warcutil.Extracted) error {
				if e.StatusCode != http.StatusOK {
					return nil
				}
				log.Println(i, e.StatusCode, e.Size, e.URI)
				return nil
			})
			extractor := warcutil.Extractor{
				Filters: []warcutil.ResponseFilter{
					warcutil.PDFResponseFilter,
				},
				Processors: []warcutil.Processor{
					debugProcessor,
					// httpPostProcessor,
					// &warcutil.HashDirProcessor{
					// 	Dir:       appCacheDir,
					// 	Extension: ".pdf",
					// },
				},
			}
			if *postURL != "" {
				var httpPostProcessor = &warcutil.HttpPostProcessor{
					URL: *postURL,
				}
				extractor.Processors = append(extractor.Processors, httpPostProcessor)
			}
			if err := extractor.Extract(resp.Body); err != nil {
				log.Fatal(err)
			}
		}
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
