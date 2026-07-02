package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// ── Blocked file extensions ───────────────────────────────────────────────────

var blockedExtensions = map[string]bool{
	".exe": true, ".bat": true, ".cmd": true, ".com": true,
	".dll": true, ".sys": true, ".drv": true, ".ocx": true,
	".sh":  true, ".bash": true, ".zsh": true, ".fish": true, ".csh": true,
	".ps1": true, ".psm1": true, ".psd1": true,
	".vbs": true, ".vbe": true, ".js":  true, ".jse": true, ".wsf": true, ".wsh": true,
	".msi": true, ".msp": true, ".msu": true,
	".scr": true, ".pif": true, ".reg": true,
	".jar": true, ".class": true,
	".hta": true, ".htm": true, ".html": true,
}

// ── Blocked MIME types (detected from actual file bytes) ─────────────────────

var blockedMIMEPrefixes = []string{
	"application/x-dosexec",
	"application/x-executable",
	"application/x-msdos-program",
	"application/x-msdownload",
	"application/x-sh",
	"application/x-shellscript",
	"application/x-msi",
	"application/x-jar",
	"application/java-archive",
}

// ValidateFileType checks the file extension against the blocklist and verifies
// the actual file bytes match a safe MIME type. Returns an error if rejected.
func ValidateFileType(filename string, data []byte) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if blockedExtensions[ext] {
		return fmt.Errorf("file type %q is not permitted for upload", ext)
	}

	// Detect real MIME from content (first 512 bytes)
	sniffBuf := data
	if len(sniffBuf) > 512 {
		sniffBuf = sniffBuf[:512]
	}
	detectedMIME := http.DetectContentType(sniffBuf)

	for _, blocked := range blockedMIMEPrefixes {
		if strings.HasPrefix(detectedMIME, blocked) {
			return fmt.Errorf("file content type %q is not permitted for upload", detectedMIME)
		}
	}

	return nil
}

// ── ClamAV INSTREAM scanner ───────────────────────────────────────────────────
//
// Implements the ClamAV INSTREAM protocol over TCP without external libraries.
// Protocol spec:
//   1. Send:  "nINSTREAM\n"
//   2. Send:  4-byte big-endian chunk length + chunk bytes (repeat)
//   3. Send:  4 zero bytes to signal end-of-stream
//   4. Read:  "stream: OK\n" or "stream: <VIRUS> FOUND\n" or "stream: ERROR ...\n"

// ScanClamAV connects to the clamd TCP daemon at addr (e.g. "clamd:3310"),
// streams data, and returns a non-nil error if a threat is found or the scan fails.
// If the connection fails (clamd not running), it returns an error — callers decide
// whether to fail-open or fail-closed based on the CLAMAV_REQUIRED env var.
func ScanClamAV(addr string, data []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("clamd unavailable at %s: %w", addr, err)
	}
	defer conn.Close()

	// Set a generous deadline for large file streams (30s)
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return fmt.Errorf("clamd: failed to set deadline: %w", err)
	}

	// 1. Send INSTREAM command
	if _, err := fmt.Fprint(conn, "nINSTREAM\n"); err != nil {
		return fmt.Errorf("clamd: failed to send command: %w", err)
	}

	// 2. Stream the file in one chunk (clamd handles up to StreamMaxLength, default 100 MB)
	chunk := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(data)))
	copy(chunk[4:], data)
	if _, err := conn.Write(chunk); err != nil {
		return fmt.Errorf("clamd: failed to stream data: %w", err)
	}

	// 3. End-of-stream marker
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return fmt.Errorf("clamd: failed to send EOS: %w", err)
	}

	// 4. Read response
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("clamd: failed to read response: %w", err)
	}
	response := strings.TrimSpace(string(buf[:n]))

	if strings.Contains(response, "FOUND") {
		// "stream: Eicar-Signature FOUND" → extract threat name
		parts := strings.SplitN(response, ": ", 2)
		threat := "unknown threat"
		if len(parts) == 2 {
			threat = strings.TrimSuffix(parts[1], " FOUND")
		}
		return fmt.Errorf("threat detected: %s", threat)
	}

	if strings.Contains(response, "ERROR") {
		return fmt.Errorf("clamd scan error: %s", response)
	}

	return nil // clean
}
