package rag

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText returns plain text from the file bytes based on its extension.
// Supported: pdf, docx, txt, and all plain-text/code types.
func ExtractText(data []byte, ext string) (string, error) {
	switch strings.ToLower(ext) {
	case "pdf":
		return extractPDF(data)
	case "docx":
		return extractDOCX(data)
	case "txt", "md", "csv",
		"js", "jsx", "ts", "tsx",
		"py", "go", "java", "c", "cpp", "cs", "php", "sh",
		"html", "css", "json", "xml", "yaml", "yml", "sql":
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported file type for text extraction: %s", ext)
	}
}

// extractPDF reads a PDF's text content page by page.
func extractPDF(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pdf open: %w", err)
	}

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteByte('\n')
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("no extractable text found in PDF (may be scanned/image-only)")
	}
	return result, nil
}

// extractDOCX reads word/document.xml from the DOCX ZIP archive.
func extractDOCX(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docx open: %w", err)
	}

	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("docx read xml: %w", err)
		}
		defer rc.Close()

		raw, err := io.ReadAll(rc)
		if err != nil {
			return "", fmt.Errorf("docx read bytes: %w", err)
		}
		return xmlToText(raw), nil
	}

	return "", fmt.Errorf("word/document.xml not found in DOCX")
}

// xmlToText strips all XML tags and returns the inner text content.
func xmlToText(xmlData []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	var sb strings.Builder
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if ch, ok := tok.(xml.CharData); ok {
			text := strings.TrimSpace(string(ch))
			if text != "" {
				sb.WriteString(text)
				sb.WriteByte(' ')
			}
		}
	}
	return strings.TrimSpace(sb.String())
}
