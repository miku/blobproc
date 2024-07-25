# blobrun

status: not implemented, just a sketch and notes

A webhook server that can receive raw bytes and execute commands. Original use
case: Receiving scholarly PDF documents and running a few derivations on them.

This service does not implement any generic features for now.

## Mode of operation

blobrun saves all incoming files in an *spool* folder and then returns, so this
processing should not take longer than the time it takes to write the file to
disk.

A periodic scan of the *spool* directory will pick up new files, and
will process them, e.g. send the content to grobid, run pdftotext, and similar.

These derivations can fail and retried, there is not time pressure, as long as
the "spool" directory does not exceed a given limit, e.g. 80% of the free space
on the disk.

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
