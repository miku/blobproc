package fetchutils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	warc "github.com/internetarchive/gowarc"
)

type WarcFilterFunc func(r *warc.Record) bool

type WarcFetch struct {
	Location string
	Filter   WarcFilterFunc
}

func (wf *WarcFetch) Run() error {
	if wf.Location == "" {
		return nil
	}
	var r io.Reader
	switch {
	case strings.HasPrefix(wf.Location, "http"):
		f, err := os.CreateTemp("", "blobfetch-warc-temp-*")
		if err != nil {
			return err
		}
		if err := Download(f.Name(), wf.Location); err != nil {
			return err
		}
		log.Printf("fetched file: %v", f.Name())
		r = f
	default:
		f, err := os.Open(wf.Location)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	reader, err := warc.NewReader(r)
	if err != nil {
		return err
	}
	for {
		record, err := reader.ReadRecord()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if record.Header.Get("WARC-Type") != "response" {
			continue
		}
		if shouldProcess(record) {
			fmt.Print("✅ ")
		} else {
			fmt.Print("   ")
		}
		fmt.Printf("[%s]: %s\n", record.Header.Get("WARC-Type"), record.Header.Get("WARC-Target-URI"))
	}
	return nil
}

func shouldProcess(record *warc.Record) bool {
	if record.Header.Get("WARC-Type") != "response" {
		return false
	}
	content := record.Content
	if content == nil {
		return false
	}
	if seeker, ok := content.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}
	scanner := bufio.NewScanner(content)
	if !scanner.Scan() {
		return false
	}
	statusLine := scanner.Text()
	if !strings.Contains(statusLine, "200") && !strings.Contains(statusLine, "HTTP/1.1 200") && !strings.Contains(statusLine, "HTTP/1.0 200") {
		return false
	}
	contentType := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line == "\r" {
			// End of headers
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-type:") {
			contentType = strings.ToLower(strings.TrimSpace(line[13:]))
			break
		}
	}
	return strings.Contains(contentType, "application/pdf") ||
		strings.Contains(contentType, "pdf")
}

func Download(dst string, url string) (err error) {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
