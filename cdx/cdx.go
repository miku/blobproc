// Package cdx wraps CDX records.
package cdx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

var ErrParsingFailed = errors.New("cdx parsing failed")

// New returns a new File which allows to access records.
func New(loc string) *File {
	switch {
	case strings.HasPrefix(loc, "http"):
	default:
		// TODO: assume local file path
	}
	// TODO: if remote, cache a copy and use that
	// curl -vL -r 127643789-128007786
	// https://archive.org/download/OJS-SITEMAP-PATCH-CRAWL-2024-07-20240823202754595-00000-00053-wbgrp-crawl666/OJS-SITEMAP-PATCH-CRAWL-2024-07-20240823222729780-00025-1703702~wbgrp-crawl666.us.archive.org~8443.warc.gz
	// -o x.pdf.gz
	return nil
}

// Record is a subset of fields from a CDX line. TODO: build this out to a full
// CDX parser. Format documetation:
// https://iipc.github.io/warc-specifications/specifications/cdx-format/cdx-2015/.
// Defaults: CDX N b a m s k r M S V g. Example:
// 30,50,51,193)/favicon.ico 20170807235758 http://193.51.50.30/favicon.ico text/html 404 OQZG7JRK66WRSYE2XJWDQ53JJYH7K44S - - 562 543915129 MSAG-PDF-CRAWL-2017-08-04-20170807232818704-00000-00009-wbgrp-svc284/MSAG-PDF-CRAWL-2017-08-04-20170807235601196-00006-3480~wbgrp-svc284.us.archive.org~8443.warc.gz
type Record struct {
	URL                  string // [2]
	MimeType             string // [3]
	ResponseCode         int    // [4]
	CompressedRecordSize int    // [8]
	CompressedOffset     int    // [9]
	Filename             string // [10]
}

// ParseRecord parses a line into a record. Default heritrix fields for the
// moment: CDX N b a m s k r M S V g
func ParseRecord(line string) (*Record, error) {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return nil, ErrParsingFailed
	}
	var err error
	record := &Record{
		URL:      fields[2],
		MimeType: fields[3],
		Filename: fields[10],
	}
	if record.ResponseCode, err = strconv.Atoi(fields[4]); err != nil {
		return nil, err
	}
	if record.CompressedRecordSize, err = strconv.Atoi(fields[8]); err != nil {
		return nil, err
	}
	if record.CompressedOffset, err = strconv.Atoi(fields[9]); err != nil {
		return nil, err
	}
	return record, nil
}

// File is a CDX file.
type File struct {
	Path string
}

// Doer is a minimal http client surface.
type Doer interface {
	Do(req http.Request) (resp http.Response, err error)
}

// WaybackFetcher can fetch the blob for a given CDX record efficiently with
// range requests. TODO: May want a file fetcher as well, as we want to test
// locally.
type WaybackFetcher struct {
	Server string
	Client Doer
}

// Fetch fetches the actual blob from wayback with range requests.
func (f *WaybackFetcher) Fetch(record *Record) ([]byte, error) {
	return nil, nil
}
