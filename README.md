# BLOBPROC

> Queues like it's 1995!

![](static/00741.png)

BLOBPROC is a less kafkaesque version of PDF postprocessing found in
sandcrawler, which is part of [IA Scholar](https://scholar.archive.org) infra.
Specifically it is designed to process and persist documents with minimum
number of external components and little to no state.

The goal is to have artifacts (fulltext, thumbnails, metadata, ...)  derived
from millions of PDF files available in a storage system (e.g. S3). In the best
case, the artifacts can be kept up to date in an unattended way.

BLOBPROC currently ships with two cli programs:

* **blobprocd** exposes an HTTP server that can receive binary data and stores
  it in a
  [spool](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s14.html) folder (maybe a better name would be `blob-spoold`)
* **blobproc** is a process that scans the spool folder and executes post
  processing tasks on each PDF, and removes the file from spool, if a
  best-effort-style processing of the file is done (periodically called by a
  systemd timer) (this is a one off command, not a server)

In our case pdf data may come from:

* Heritrix crawl, via a [ScriptedProcessor](https://github.com/miku/blobproc/blob/8e9f091ea83c46b024b0c74ee7900b1fb84c4174/extra/heritrix/fetch-processor-snippet.xml#L30-L137)
* (wip) a WARC file, a crawl collection or similar
* in general, by any process that can deposit a file in the spool folder or send an HTTP request to blobprocd

In our case blobproc will execute the following tasks:

* send PDF to [GROBID](https://github.com/kermitt2/grobid) and store the result in **S3**, using [grobidclient](https://github.com/miku/grobidclient) Go library
* generate text from PDF via [pdftotext](https://www.xpdfreader.com/pdftotext-man.html) and store the result in S3 ([seaweedfs](https://github.com/seaweedfs/seaweedfs))
* generate a thumbnail from PDF via [pdftoppm](https://www.xpdfreader.com/pdftoppm-man.html) and store the result in S3 ([seaweedfs](https://github.com/seaweedfs/seaweedfs))
* find all weblinks in the PDF text and send them to a crawl API (wip)

More tasks can be added by extending blobproc itself. A focus remains on simple
deployment via an OS distribution package. By pushing various parts into library
functions (or external packages like [grobidclient](https://miku/grobidclient)), the main processing routine shrinks to about [100 lines of
code](https://github.com/miku/blobproc/blob/37f9cd7873f1e08400f46e98640e2b24bd37a088/walker.go#L64-L166)
(as of 08/2024). Currently both blobproc and blobprocd run on a dual-core [2nd
gen
XEON](https://ark.intel.com/content/www/us/en/ark/products/193394/intel-xeon-silver-4216-processor-22m-cache-2-10-ghz.html) with 24GB of RAM;
blobprocd received up to 100 rps and wrote pdfs to rotational disk.

## Bulk, back-of-the-envelope, reprocessing

Currently, about 5 pdfs/s. GROBID may be able to handle up to 10 pdfs/s. To
reprocess, say 200M pdfs in less than a month, we would need about 10 GROBID
instances.

## Mode of operation

* receive blob over HTTP, may be heritrix, curl, some backfill process
* regularly scan spool dir and process found files

## Usage

Server component.

```
$ blobprocd -h
Usage of blobprocd:
  -T duration
        server timeout (default 15s)
  -access-log string
        server access logfile, none if empty
  -addr string
        host port to listen on (default "0.0.0.0:8000")
  -debug
        switch to log level DEBUG
  -log string
        structured log output file, stderr if empty
  -spool string
         (default "/home/tir/.local/share/blobproc/spool")
  -version
        show version
```

Processing command line tool.

```
$ blobproc --help
BLOBPROC is a PDF postprocessing utility that generates derivatives
like fulltext, thumbnails, and metadata from PDF files and can persist them to S3.

Examples:
  blobproc run                    # Process files from spool directory (sequential)
  blobproc run -w 4               # Process with 4 parallel workers
  blobproc single file.pdf        # Process single file for testing
  blobproc config                 # Show current configuration

Usage:
  blobproc [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Show current configuration
  help        Help about any command
  run         Process files from spool directory
  single      Process a single file for testing

Flags:
      --config string      config file (searches: ./blobproc.yaml, /home/tir/.config/blobproc/blobproc.yaml, /etc/blobproc/blobproc.yaml)
      --debug              enable debug logging
  -h, --help               help for blobproc
      --log-file string    log file path (empty = stderr)
      --spool-dir string   spool directory path (default "/home/tir/.local/share/blobproc/spool")
      --timeout duration   subprocess timeout (default 5m0s)
  -v, --version            version for blobproc

Use "blobproc [command] --help" for more information about a command.
```

## Performance data points

The initial, unoptimized version would process about 25 pdfs/minute or 36K
pdfs/day. We were able to crawl much faster than that, e.g. we reached 63G
captured data (not all pdf) after about 4 hours. GROBID should be able to
handle up to 10 pdfs/s.

A parallel walker could process about 300 pdfs/minute, and would match the
inflow generated by one heritrix crawl node.

## Scaling

* [x] tasks will run in parallel, e.g. text, thumbnail generation and grobid all run in parallel, but we process one file by one for now
* [ ] we should be able to configure a pool of grobid hosts to send requests to

## Backfill

* [ ] point to CDX file, crawl collection or similar and have all PDF files sent to BLOBPROC, even if this may take days or weeks

## TODO

* [ ] for each file placed into spool, try to record the URL-SHA1 pair somewhere
* [ ] pluggable write backend for testing, e.g. just log what would happen
* [ ] log performance measures
* [ ] grafana

## Notes

This tool should cover most of the following areas from sandcrawler:

* `run_grobid_extract`
* `run_pdf_extract`
* `run_persist_grobid`
* `run_persist_pdftext`
* `run_persist_thumbnail`

Including references workers.

Performance: Processing 1605 pdfs, 1515 successful, 2.23 docs/s, when processed
in parallel, via `fd ... -x` - or about 200K docs per day.

```
real    11m0.767s
user    73m57.763s
sys     5m55.393s
```

