// blobfeed can feed blobprocd from files, WARCs, items and collections. This
// tool can be used to backfill pdf postprocessing items.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/miku/blobproc"
	"github.com/miku/blobproc/dedent"
)

var (
	server         = flag.String("s", "http://localhost:9444", "blobprocd server")
	sendFile       = flag.String("f", "", "pdf file or url to send to blobprocd")
	sendWarc       = flag.String("w", "", "send all pdfs found in a WARC file to blobproc")
	sendCdx        = flag.String("x", "", "send all pdfs found in a CDX file")
	sendItem       = flag.String("i", "", "send all pdfs found in all WARC files from an item")
	sendCollection = flag.String("c", "", "send all pdfs found in all WARC files found in items")
	timeout        = flag.Duration("T", 30*time.Second, "timeout")
	verbose        = flag.Bool("v", false, "verbose output")
)

func main() {
	flag.Parse()
	spoolURL, err := url.JoinPath(*server, "/spool")
	if err != nil {
		log.Fatal(err)
	}
	switch {
	case *sendFile != "":
		if _, err := exec.LookPath("curl"); err != nil {
			log.Fatal("curl is required")
		}
		curlOpts := fmt.Sprintf(`--retry-max-time %d --retry 3`, int(timeout.Seconds()))
		var c string // command string
		switch {
		case strings.HasPrefix(*sendFile, "http"):
			c = dedent.Sprintf(`
				curl %s -s "%s" | curl %s -XPOST --data-binary @- -H 'User-Agent: blobfeed/%s' -H 'X-BLOBPROC-URL: %s' "%s"`,
				curlOpts,
				*sendFile,
				curlOpts,
				blobproc.Version,
				*sendFile,
				spoolURL,
			)
		default:
			c = dedent.Sprintf(`
				curl %s -XPOST --data-binary @%s -H 'User-Agent: blobfeed/%s' "%s"`,
				curlOpts,
				*sendFile,
				blobproc.Version,
				*sendFile,
				spoolURL,
			)
		}
		if *verbose {
			log.Println(c)
		}
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bash", "-c", c)
		b, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("send failed")
			log.Fatal(string(b))
		}
	case *sendCdx != "":

	case *sendWarc != "":
		// parse a warc
	case *sendItem != "":
	case *sendCollection != "":
	}
}
