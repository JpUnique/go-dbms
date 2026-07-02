package rag

import "strings"

const (
	chunkSize    = 500 // approximate words per chunk
	chunkOverlap = 50  // words of overlap between consecutive chunks
)

// Chunk splits text into overlapping word-based chunks.
// Each chunk is a plain string ready to be embedded.
func Chunk(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var chunks []string
	start := 0
	for start < len(words) {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[start:end], " "))
		if end == len(words) {
			break
		}
		start += chunkSize - chunkOverlap
	}
	return chunks
}
