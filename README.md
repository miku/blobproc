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

