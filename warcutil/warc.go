package warcutil

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	warc "github.com/internetarchive/gowarc"
)

type Extracted struct {
	URI         string
	ContentType string
	Content     io.Reader
	Size        int64
	Record      *warc.Record
}

type Processor interface {
	Process(Extracted) error
}

type FuncProcessor func(Extracted) error

func (f FuncProcessor) Process(ex Extracted) error {
	return f(ex)
}

type ResponseFilter func(resp *http.Response) bool

var PDFResponseFilter = func(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/pdf")
}

var DebugProcessor = FuncProcessor(func(e Extracted) error {
	log.Println(e.URI)
	return nil
})

// DirProcessor writes extracted files into a given directory.
type DirProcessor struct {
	Dir       string
	Prefix    string
	Extension string
}

func (d *DirProcessor) Process(ex Extracted) error {
	f, err := os.CreateTemp(d.Dir, fmt.Sprintf("%s*%s", d.Prefix, d.Extension))
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, ex.Content); err != nil {
		return err
	}
	return nil
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type HttpPostProcessor struct {
	URL    string
	Client Doer
}

func (h *HttpPostProcessor) Process(ex Extracted) error {
	if h.Client == nil {
		h.Client = http.DefaultClient
	}
	buf := bytes.Buffer{}
	if _, err := io.Copy(&buf, ex.Content); err != nil {
		return err
	}
	req, err := http.NewRequest("POST", h.URL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", ex.ContentType)
	req.Header.Set("X-BLOBPROC-URL", ex.URI)

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %v", resp.StatusCode)
	}
	return nil
}

type Extractor struct {
	Filters    []ResponseFilter
	Processors []Processor
}

func (e *Extractor) Extract(r io.Reader) error {
	wr, err := warc.NewReader(r)
	if err != nil {
		return err
	}
	for {
		record, err := wr.ReadRecord()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		uri := record.Header.Get("WARC-Target-URI")
		if len(uri) == 0 {
			continue
		}
		warcContentType := record.Header.Get("Content-Type")
		if warcContentType != "application/http; msgtype=response" {
			continue
		}
		l := record.Header.Get("Content-Length")
		contentLength, err := strconv.ParseInt(l, 10, 64)
		if err != nil {
			continue
		}
		limitedReader := io.LimitReader(record.Content, contentLength)
		resp, err := http.ReadResponse(bufio.NewReader(limitedReader), nil)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		shouldProcess := true
		for _, filter := range e.Filters {
			if ok := filter(resp); !ok {
				shouldProcess = false
				break
			}
		}
		if !shouldProcess {
			_ = resp.Body.Close()
			continue
		}
		extracted := Extracted{
			URI:         uri,
			ContentType: resp.Header.Get("Content-Type"),
			Content:     resp.Body,
			Size:        resp.ContentLength,
			Record:      record,
		}
		for _, processor := range e.Processors {
			if err := processor.Process(extracted); err != nil {
				return err
			}
		}
	}
	return nil
}
