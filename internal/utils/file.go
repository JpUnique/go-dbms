package utils

import (
	"fmt"
	"time"
)

// GenerateFileKey creates unique file key for MinIO
func GenerateFileKey(filename string) string {
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
}
