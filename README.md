# BLOBPROC

status: testing

BLOBPROC is a *shrink wrap* version of PDF blob postprocessing found in
sandcrawler. Specifically it is designed to process and persist documents
*without any extra component*, like a database or a separate queuing system and
do this in a performant, reliant, boring and observable way.

BLOBPROC contains two components:

* **blobprocd** exposes an *HTTP server* that can receive binary data and stores it in a [spool](https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch05s14.html) folder
* **blobproc** is a process that scans the spool folder and executes post processing tasks on each PDF, and removes the file from spool, if all processing succeeded

In our case blobproc will execute the following tasks:

* send PDF to grobid and store the result in S3
* generate text from PDF and store the result in S3
* generate a thumbnail from a PDF and store the result in S3
* find all weblinks in PDF text and send them to a crawl API

More tasks can be added by extending blobproc itself. A focus remains on simple
deployment via an OS distribution package.

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
$ blobproc -h
blobproc - process and persist PDF documents derivations

Emit JSON with locally extracted data:

  $ blobproc -f file.pdf | jq .

Flags

  -T duration
        subprocess timeout (default 5m0s)
  -debug
        more verbose output
  -f string
        process a single file (local tools only), for testing
  -grobid-host string
        grobid host, cf. https://is.gd/3wnssq (default "http://localhost:8070")
  -grobid-max-filesize int
        max file size to send to grobid in bytes (default 268435456)
  -k    keep files in spool after processing, mainly for debugging
  -logfile string
        structured log output file, stderr if empty
  -s3-access-key string
        S3 access key (default "minioadmin")
  -s3-endpoint string
        S3 endpoint (default "localhost:9000")
  -s3-secret-key string
        S3 secret key (default "minioadmin")
  -spool string
         (default "/home/tir/.local/share/blobproc/spool")
  -version
        show version
```

## Performance data points

The initial, unoptimized version would process about 25 PDF docs/minute or 36K
pdfs/day. We were able to crawl much faster than that, e.g. we reached 63G
captured data (not all pdf) after about 4 hours. GROBID should be able to
handle up to 10 docs/s.

## Scaling

* TODO: tasks will run in parallel, e.g. text, thumbnail generation and grobid all run in parallel, but we process one file by one for now
* TODO: we should be able to configure a pool of grobid hosts to send requests to

## Backfill

* point to CDX file, crawl collection or similar and have all PDF files sent to BLOBPROC, even if this may take days or weeks

## TODO

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

