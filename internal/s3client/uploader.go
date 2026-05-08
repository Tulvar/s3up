package s3client

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	_, err = u.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
		Body:   file,
	})
	return err
}
