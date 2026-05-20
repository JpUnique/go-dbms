package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	client     *minio.Client
	bucketName string
)

// ==============================
// INIT MINIO (CALLED IN main.go)
// ==============================
func InitMinIO() error {

	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucketName = os.Getenv("MINIO_BUCKET")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucketName == "" {
		return errors.New("minio configuration missing")
	}

	var err error

	client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("minio init: %w", err)
	}

	// Check if bucket exists
	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return fmt.Errorf("minio check bucket: %w", err)
	}

	//  Create bucket if not exists
	if !exists {

		err = client.MakeBucket(
			context.Background(),
			bucketName,
			minio.MakeBucketOptions{},
		)
		if err != nil {
			return fmt.Errorf("minio create bucket: %w", err)
		}
	}

	return nil
}

// ==============================
// GENERATE FILE KEY
// ==============================
func GenerateFileKey(fileName string) string {

	ext := ""
	if i := strings.LastIndex(fileName, "."); i != -1 {
		ext = fileName[i:]
	}

	return uuid.NewString() + ext
}

// ==============================
// UPLOAD FILE
// ==============================
func Upload(fileKey string, data []byte, contentType string) error {

	if client == nil {
		return errors.New("minio client not initialized")
	}

	reader := bytes.NewReader(data)

	_, err := client.PutObject(
		context.Background(),
		bucketName,
		fileKey,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		return fmt.Errorf("minio upload: %w", err)
	}

	return nil
}

// ==============================
// GET DOWNLOAD URL (PRESIGNED)
// ==============================
func GetDownloadURL(fileKey string) (string, error) {

	if client == nil {
		return "", errors.New("minio client not initialized")
	}

	reqParams := make(url.Values)

	url, err := client.PresignedGetObject(
		context.Background(),
		bucketName,
		fileKey,
		time.Minute*10,
		reqParams,
	)

	if err != nil {
		return "", fmt.Errorf("minio get url: %w", err)
	}

	return url.String(), nil
}

// ==============================
// DELETE FILE
// ==============================
func Delete(fileKey string) error {

	if client == nil {
		return errors.New("minio client not initialized")
	}

	err := client.RemoveObject(
		context.Background(),
		bucketName,
		fileKey,
		minio.RemoveObjectOptions{},
	)

	if err != nil {
		return fmt.Errorf("minio delete: %w", err)
	}

	return nil
}
