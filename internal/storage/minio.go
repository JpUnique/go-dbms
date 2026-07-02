package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var client *minio.Client

// ======================================
// INIT MINIO
// ======================================
func InitMinIO() error {

	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")

	if endpoint == "" || accessKey == "" || secretKey == "" {
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

	fmt.Println("✅ MinIO initialized")

	return nil
}

// ======================================
// GENERATE USER BUCKET NAME
// ======================================
func getUserBucket(username string) string {
	return "user-" + strings.ToLower(username)
}

// ======================================
// ENSURE BUCKET EXISTS
// ======================================
func EnsureUserBucket(ctx context.Context, username string) error {

	bucket := getUserBucket(username)

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}

	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("create bucket failed: %w", err)
		}
		fmt.Println("✅ Bucket created:", bucket)
	}

	return nil
}

// ======================================
// GENERATE FILE KEY
// ======================================
func GenerateFileKey(fileName string) string {

	ext := ""
	if i := strings.LastIndex(fileName, "."); i != -1 {
		ext = fileName[i:]
	}

	return uuid.NewString() + ext
}

// ======================================
// UPLOAD FILE (PER USER)
// ======================================
func Upload(username, fileKey string, data []byte, contentType string) error {

	if client == nil {
		return errors.New("minio client not initialized")
	}

	ctx := context.Background()
	bucket := getUserBucket(username)

	//  ensure bucket exists
	if err := EnsureUserBucket(ctx, username); err != nil {
		return err
	}

	reader := bytes.NewReader(data)

	_, err := client.PutObject(
		ctx,
		bucket,
		fileKey,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	return nil
}

// ======================================
// GET DOWNLOAD URL
// ======================================
func GetDownloadURL(username, fileKey string) (string, error) {

	if client == nil {
		return "", errors.New("minio client not initialized")
	}

	ctx := context.Background()
	bucket := getUserBucket(username)

	reqParams := make(url.Values)

	u, err := client.PresignedGetObject(
		ctx,
		bucket,
		fileKey,
		time.Minute*10,
		reqParams,
	)

	if err != nil {
		return "", fmt.Errorf("get url failed: %w", err)
	}

	return u.String(), nil
}

// ======================================
// GET FILE BYTES
// ======================================
func GetFileBytes(username, fileKey string) ([]byte, string, error) {
	if client == nil {
		return nil, "", errors.New("minio client not initialized")
	}

	ctx := context.Background()
	bucket := getUserBucket(username)

	obj, err := client.GetObject(ctx, bucket, fileKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("get object failed: %w", err)
	}
	defer obj.Close()

	info, err := obj.Stat()
	if err != nil {
		return nil, "", fmt.Errorf("stat object failed: %w", err)
	}

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, "", fmt.Errorf("read object failed: %w", err)
	}

	return data, info.ContentType, nil
}

// ======================================
// DELETE FILE
// ======================================
func Delete(username, fileKey string) error {

	if client == nil {
		return errors.New("minio client not initialized")
	}

	ctx := context.Background()
	bucket := getUserBucket(username)

	err := client.RemoveObject(
		ctx,
		bucket,
		fileKey,
		minio.RemoveObjectOptions{},
	)

	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}
