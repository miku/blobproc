package cdx

// New returns a new File which allows to access records.
func New(loc string) *File {
	// TODO: if remote, cache a copy and use that
	// curl -vL -r 127643789-128007786
	// https://archive.org/download/OJS-SITEMAP-PATCH-CRAWL-2024-07-20240823202754595-00000-00053-wbgrp-crawl666/OJS-SITEMAP-PATCH-CRAWL-2024-07-20240823222729780-00025-1703702~wbgrp-crawl666.us.archive.org~8443.warc.gz
	// -o x.pdf.gz
	return nil
}

// File is a CDX file
type File struct {
}
