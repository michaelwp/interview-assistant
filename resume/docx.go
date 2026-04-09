package resume

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

func readDocx(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("open DOCX: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			return extractDocxText(rc)
		}
	}
	return "", fmt.Errorf("invalid DOCX: word/document.xml not found")
}

// extractDocxText pulls plain text out of a DOCX word/document.xml stream.
func extractDocxText(r io.Reader) (string, error) {
	var sb strings.Builder
	dec := xml.NewDecoder(r)
	inText := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "p":
				sb.WriteString("\n")
			case "br", "cr":
				sb.WriteString("\n")
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
		case xml.CharData:
			if inText {
				sb.Write(t)
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}
