# BLOBPROC

status: testing

For an influx of PDF files, we wanted to have a small, event-based component
that would apply processing to those files, using a hotfolder.

The server receives raw bytes in the HTTP body, stores them in a folder. Another processes regularly scans this folder, and executes commands. Original use
case: Receiving scholarly PDF documents from heritrix, then running and storing derivations.

This service does not implement any generic features for now.

![](static/00596.png)

## Mode of operation

blobrun saves all incoming files in an *spool* folder and then returns, so this
processing should not take longer than the time it takes to write the file to
disk.

A periodic scan of the *spool* directory will pick up new files, and
will process them, e.g. send the content to grobid, run pdftotext, and similar.

These derivations can fail and retried, there is not time pressure, as long as
the "spool" directory does not exceed a given limit.

Once all derivations ran successfully, the file is deleted from the "spool"
directory. If the server dies and comes back up, the files in the "spool"
directory represent the state.

## Derivations

For S3, the key will be the content SHA1.

* pdftotext, store in S3
* grobid, store in S3
* thumbnail, store in S3
* find all links in fulltext, send to SPNv2

## Backfill

Given a cli tool to fetch a list of PDFs from PB, we can complete missing
derivations.
