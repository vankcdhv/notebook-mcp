package notebooklm

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"
)

// Deep research carries its generated report inside a base64 protobuf blob.
// The blob is a sequence of field 1 entries; each holds the research plan
// (field 1), a consulted source (field 2), or the report itself (field 3).
// The report's field 1 is the markdown body, whose first heading is the title
// NotebookLM shows once the report is imported as a source.
const (
	reportEntryField  = 1
	reportBodyField   = 3
	reportMarkdownTag = 1
)

// decodeResearchReport extracts the markdown body from a deep research blob.
// It returns an empty string while the report is still being generated.
func decodeResearchReport(encoded string) string {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return ""
		}
	}
	for _, entry := range protoFields(raw, reportEntryField) {
		for _, body := range protoFields(entry, reportBodyField) {
			for _, markdown := range protoFields(body, reportMarkdownTag) {
				if utf8.Valid(markdown) {
					return string(markdown)
				}
			}
		}
	}
	return ""
}

// reportTitle reads the leading markdown heading, falling back to the first
// non-empty line so an unheaded report still gets a usable source name.
func reportTitle(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.TrimSpace(strings.TrimLeft(line, "#"))
	}
	return ""
}

// protoFields returns the payloads of every length-delimited field matching
// number, ignoring the other wire types the blob mixes in.
func protoFields(data []byte, number uint64) [][]byte {
	var out [][]byte
	for pos := 0; pos < len(data); {
		tag, n := protoVarint(data[pos:])
		if n == 0 {
			return out
		}
		pos += n
		switch tag & 7 {
		case 0:
			_, n := protoVarint(data[pos:])
			if n == 0 {
				return out
			}
			pos += n
		case 1:
			pos += 8
		case 2:
			length, n := protoVarint(data[pos:])
			if n == 0 || pos+n+int(length) > len(data) {
				return out
			}
			pos += n
			if tag>>3 == number {
				out = append(out, data[pos:pos+int(length)])
			}
			pos += int(length)
		case 5:
			pos += 4
		default:
			return out
		}
	}
	return out
}

func protoVarint(data []byte) (uint64, int) {
	var value uint64
	var shift uint
	for i, b := range data {
		if i > 9 {
			return 0, 0
		}
		if b < 0x80 {
			return value | uint64(b)<<shift, i + 1
		}
		value |= uint64(b&0x7f) << shift
		shift += 7
	}
	return 0, 0
}
