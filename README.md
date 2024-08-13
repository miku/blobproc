# BLOBPROC

status: testing

BLOBPROC contains two components:

* **blobprocd** exposes an HTTP server that can receive binary data (here: PDF) and stores it into a spool folder
* **blobproc** is a process that scans the spool folder and executes post processing tasks on each PDF, and removes the file from spool, if all processing succeeded

In our case blobproc will execute the following tasks:

* send PDF to grobid and store the result in S3
* generate text from PDF and store the result in S3
* generate a thumbnail from a PDF and store the result in S3
* find all weblinks in PDF text and send them to a crawl API

More tasks can be added by extending blobproc itself. Our focus is on simple deployment.

![](static/00596.png)

## Mode of operation

* receive blob over HTTP, may be heritrix, curl, some backfill process
* regularly scan spool dir and process found files

## Scaling

* tasks will run in parallel, e.g. text, thumbnail generation and grobid all run in parallel, but we process one file by one for now
* we should be able to configure a pool of grobid hosts to send requests to

## Backfill

* point to CDX file, crawl collection or similar and have all PDF files sent to BLOBPROC, even if this may take days or weeks

## TODO

* [ ] pluggable write backend for testing, e.g. just log what would happen
* [ ] log performance measures
* [ ] grafana

----

Image credit: [SD](https://github.com/CompVis/stable-diffusion)
