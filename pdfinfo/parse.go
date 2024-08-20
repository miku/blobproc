package pdfinfo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Metadata groups output of various tools into a single struct.
type Metadata struct {
	PDFCPU  *PDFCPU `json:"pdfcpu,omitempty"`  // pdfcpu output, parsed into JSON.
	PDFInfo *Info   `json:"pdfinfo,omitempty"` // pdfinfo, parsed into JSON.
}

func (metadata Metadata) LegacyPDFExtra() *PDFExtra {
	return &PDFExtra{
		Page0Height: metadata.PDFInfo.PageDim().Height,
		Page0Width:  metadata.PDFInfo.PageDim().Width,
		PageCount:   metadata.PDFInfo.Pages,
		PDFVersion:  metadata.PDFInfo.PDFVersion,
	}
}

// PDFExtra was a free form dictionary in sandcrawler. Keep this here for
// compatibility.
//
// In [10]: pdf_document.pdf_id
// Out[10]: PDFId(permanent_id='070262676b9d8a3776b3a9e2c168f961',
// update_id='29245f594c8bea0fc7f2cc90ca1dd021')
type PDFExtra struct {
	Page0Height float64 `json:"page0height,omitempty"`  // in pts, we can parse "pdfinfo" output
	Page0Width  float64 `json:"page0width,omitempty"`   // in pts, we can parse "pdfinfo" output
	PageCount   int     `json:"page_count,omitempty"`   // "pdfinfo" "Pages"
	PermanentID string  `json:"permanent_id,omitempty"` // TODO: where do we get this from?
	UpdateID    string  `json:"update_id,omitempty"`    // TODO: where do we get this from?
	PDFVersion  string  `json:"pdf_version,omitempty"`  // PDF version: 1.5, ...
}

// PDFCPU structured output from pdfcpu tool.
type PDFCPU struct {
	Header struct {
		Creation string `json:"creation"`
		Version  string `json:"version"`
	} `json:"header"`
	Infos []struct {
		AppendOnly       bool     `json:"appendOnly"`
		Author           string   `json:"author"`
		Bookmarks        bool     `json:"bookmarks"`
		CreationDate     string   `json:"creationDate"`
		Creator          string   `json:"creator"`
		Encrypted        bool     `json:"encrypted"`
		Form             bool     `json:"form"`
		Hybrid           bool     `json:"hybrid"`
		Keywords         []string `json:"keywords"`
		Linearized       bool     `json:"linearized"`
		ModificationDate string   `json:"modificationDate"`
		Names            bool     `json:"names"`
		PageCount        int64    `json:"pageCount"`
		PageMode         string   `json:"pageMode"`
		PageSizes        []struct {
			Height float64 `json:"height"`
			Width  float64 `json:"width"`
		} `json:"pageSizes"`
		Permissions int64  `json:"permissions"`
		Producer    string `json:"producer"`
		Properties  struct {
			PTEXFullbanner string `json:"PTEX.Fullbanner"`
		} `json:"properties"`
		Signatures         bool   `json:"signatures"`
		Source             string `json:"source"`
		Subject            string `json:"subject"`
		Tagged             bool   `json:"tagged"`
		Thumbnails         bool   `json:"thumbnails"`
		Title              string `json:"title"`
		Unit               string `json:"unit"`
		UsingObjectStreams bool   `json:"usingObjectStreams"`
		UsingXRefStreams   bool   `json:"usingXRefStreams"`
		Version            string `json:"version"`
		Watermarked        bool   `json:"watermarked"`
	} `json:"infos"`
}

// Info is a parsed pdfinfo output.
type Info struct {
	Title          string `json:"title"`
	Subject        string `json:"subject"`
	Keywords       string `json:"keywords"`
	Author         string `json:"author"`
	Creator        string `json:"creator"`
	Producer       string `json:"producer"`
	CreationDate   string `json:"creation_date"`
	ModDate        string `json:"mod_date"`
	CustomMetadata bool   `json:"custom_metadata"`
	MetadataStream bool   `json:"metadata_stream"`
	Tagged         bool   `json:"tagged"`
	UserProperties bool   `json:"user_properties"`
	Suspects       bool   `json:"suspects"`
	Form           string `json:"form"`
	JavaScript     bool   `json:"javascript"`
	Pages          int    `json:"pages"`
	Encrypted      bool   `json:"encrypted"`
	PageSize       string `json:"page_size"` // 595.276 x 841.89 pts (A4)
	PageRot        int    `json:"page_rot"`
	FileSize       int    `json:"filesize"`
	Optimized      bool   `json:"optimized"`
	PDFVersion     string `json:"pdf_version"`
}

// Dim groups width and height of a page.
type Dim struct {
	Width  float64
	Height float64
}

func (info *Info) PageDim() Dim {
	if info == nil {
		return Dim{}
	}
	// 463.059 x 668.047 pts
	// 595 x 882 pts
	re := regexp.MustCompile(`(?<width>[0-9.]*)[\s]*x[\s]*(?<height>[0-9.]*)`)
	matches := re.FindStringSubmatch(info.PageSize)
	if len(matches) < 3 {
		return Dim{}
	}
	width, err := strconv.ParseFloat(matches[re.SubexpIndex("width")], 64)
	if err != nil {
		return Dim{}
	}
	height, err := strconv.ParseFloat(matches[re.SubexpIndex("height")], 64)
	if err != nil {
		return Dim{}
	}
	dim := Dim{
		Width:  width,
		Height: height,
	}
	return dim
}

// ParseFile a filename into a structured metadata object. Requires pdfinfo and pdfcpu to be installed.
func ParseFile(ctx context.Context, filename string) (*Metadata, error) {
	if _, err := exec.LookPath("pdfcpu"); err != nil {
		return nil, fmt.Errorf("missing pdfcpu executable")
	}
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		return nil, fmt.Errorf("missing pdfinfo executable")
	}
	var metadata = new(Metadata)
	info, err := runPdfInfo(ctx, filename)
	if err != nil {
		return nil, err
	}
	metadata.PDFInfo = info
	pdfcpu, err := runPdfCpu(ctx, filename)
	if err != nil {
		return nil, err
	}
	metadata.PDFCPU = pdfcpu
	return metadata, nil
}

// ParseFile parses a pdf file. Requires pdfinfo executable to be installed.
func runPdfCpu(ctx context.Context, filename string) (*PDFCPU, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdfcpu", "info", "-j", filename)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var pdfcpu PDFCPU
	if err := json.Unmarshal(buf.Bytes(), &pdfcpu); err != nil {
		return nil, err
	}
	return &pdfcpu, nil
}

// runPdfInfo parses a pdf file. Requires pdfinfo executable to be installed.
func runPdfInfo(ctx context.Context, filename string) (*Info, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdfinfo", filename)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return ParseInfo(buf.String()), nil
}

// ParseInfo pdfinfo output into an Info struct.
func ParseInfo(s string) *Info {
	info := Info{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		fields := strings.SplitN(line, ":", 2)
		if len(fields) != 2 {
			continue
		}
		fields[0] = strings.TrimSpace(fields[0])
		fields[1] = strings.TrimSpace(fields[1])
		switch fields[0] {
		case "Title":
			info.Title = fields[1]
		case "Subject":
			info.Subject = fields[1]
		case "Keywords":
			info.Keywords = fields[1]
		case "Author":
			info.Author = fields[1]
		case "Creator":
			info.Creator = fields[1]
		case "Producer":
			info.Producer = fields[1]
		case "CreationDate":
			info.CreationDate = fields[1]
		case "ModDate":
			info.ModDate = fields[1]
		case "Custom Metadata":
			info.CustomMetadata = parseBool(fields[1])
		case "Metadata Stream":
			info.MetadataStream = parseBool(fields[1])
		case "Tagged":
			info.Tagged = parseBool(fields[1])
		case "UserProperties":
			info.UserProperties = parseBool(fields[1])
		case "Suspects":
			info.Suspects = parseBool(fields[1])
		case "Form":
			info.Form = fields[1]
		case "JavaScript":
			info.JavaScript = parseBool(fields[1])
		case "Pages":
			info.Pages = parseInt(fields[1])
		case "Encrypted":
			info.Encrypted = parseBool(fields[1])
		case "Page size":
			info.PageSize = fields[1]
		case "Page rot":
			info.PageRot = parseInt(fields[1])
		case "File size":
			info.FileSize = parseAnyInt(fields[1])
		case "Optimized":
			info.Optimized = parseBool(fields[1])
		case "PDF version":
			info.PDFVersion = fields[1]
		default:
			log.Printf("ignoring pdfinfo field: %v", fields[0])
		}
	}
	return &info
}

// parseBool returns a bool from a string used in pdfinfo output, like "yes", and "no".
func parseBool(s string) bool {
	if s == "yes" {
		return true
	}
	return false
}

func parseInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		log.Println(err)
		return 0
	}
	return v
}

func parseAnyInt(s string) int {
	for _, tok := range strings.Fields(s) {
		v, err := strconv.Atoi(tok)
		if err != nil {
			continue
		}
		return v
	}
	return 0
}
