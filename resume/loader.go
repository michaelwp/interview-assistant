// Package resume extracts plain text from interviewee profile files.
// Supported formats: .txt, .md, .pdf, .docx
package resume

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Load reads the file at path and returns its plain-text content.
func Load(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md":
		return readText(path)
	case ".pdf":
		return readPDF(path)
	case ".docx":
		return readDocx(path)
	case ".doc":
		return "", fmt.Errorf(".doc is not supported; save the file as .docx or .txt first")
	default:
		return "", fmt.Errorf("unsupported file type %q; use .txt, .pdf, or .docx", ext)
	}
}
