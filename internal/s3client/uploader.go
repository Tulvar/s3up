package s3client

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/tulvar/s3up/internal/upload"
)

type Uploader struct {
	uploader *manager.Uploader
}

func (u *Uploader) Upload(ctx context.Context, input upload.UploadInput) error {
	file, err := os.Open(input.LocalPath)
	if err != nil {
		return err
	}
	defer file.Close()

	put := &s3.PutObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
		Body:   file,
	}
	if input.ContentType != "" {
		put.ContentType = aws.String(input.ContentType)
	}
	if len(input.Metadata) > 0 {
		put.Metadata = input.Metadata
	}
	if input.StorageClass != "" {
		put.StorageClass = types.StorageClass(input.StorageClass)
	}
	if input.ACL != "" {
		put.ACL = types.ObjectCannedACL(input.ACL)
	}

	_, err = u.uploader.Upload(ctx, put)
	return err
}
