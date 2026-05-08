package s3client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	deletepkg "github.com/tulvar/s3up/internal/delete"
)

type Deleter struct {
	client *s3.Client
}

func (d *Deleter) Delete(ctx context.Context, input deletepkg.DeleteInput) error {
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(input.Bucket),
		Key:    aws.String(input.Key),
	})
	return err
}
