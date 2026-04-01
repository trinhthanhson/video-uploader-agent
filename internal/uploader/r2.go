package uploader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"video-uploader-agent/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Uploader struct {
	client *s3.Client
	bucket string
}

func NewR2Uploader(cfg *config.Config) *R2Uploader {
	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(cfg.R2.Endpoint),
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.R2.AccessKeyID,
			cfg.R2.SecretAccessKey,
			"",
		),
	})

	return &R2Uploader{
		client: client,
		bucket: cfg.R2.Bucket,
	}
}

func (u *R2Uploader) Upload(filePath string, orderID string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileName := filepath.Base(filePath)

	// object key theo structure
	objectKey := fmt.Sprintf(
		"videos/%s/%s/%s",
		time.Now().Format("2006/01/02"),
		orderID,
		fileName,
	)

	_, err = u.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(objectKey),
		Body:   file,
	})

	if err != nil {
		return "", err
	}

	return objectKey, nil
}
