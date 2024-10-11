// Package cdx wraps CDX records.
package cdx

import "net/http"

// New returns a new File which allows to access records.
func New(loc string) *File {
	// TODO: if remote, cache a copy and use that
	// curl -vL -r 127643789-128007786
	// https://archive.org/download/OJS-SITEMAP-PATCH-CRAWL-2024-07-20240823202754595-00000-00053-wbgrp-crawl666/OJS-SITEMAP-PATCH-CRAWL-2024-07-20240823222729780-00025-1703702~wbgrp-crawl666.us.archive.org~8443.warc.gz
	// -o x.pdf.gz
	return nil
}

// https://iipc.github.io/warc-specifications/specifications/cdx-format/cdx-2015/
// CDX N b a m s k r M S V g
// 30,50,51,193)/favicon.ico 20170807235758 http://193.51.50.30/favicon.ico text/html 404 OQZG7JRK66WRSYE2XJWDQ53JJYH7K44S - - 562 543915129 MSAG-PDF-CRAWL-2017-08-04-20170807232818704-00000-00009-wbgrp-svc284/MSAG-PDF-CRAWL-2017-08-04-20170807235601196-00006-3480~wbgrp-svc284.us.archive.org~8443.warc.gz
type Record struct {
	URL                  string
	MimeType             string
	ResponseCode         int
	CompressedRecordSize int
	CompressedOffset     int
	Filename             string
}

// File is a CDX file.
type File struct {
	Path string
}

type Doer interface {
	Do(req http.Request) (resp http.Response, err error)
}

// Fetcher can fetch the blob for a given CDX record efficiently with range requests.
type Fetcher struct {
	Server string
	Client Doer
}

func (f *Fetcher) Fetch(record *Record) ([]byte, error) {
	return nil, nil
}
