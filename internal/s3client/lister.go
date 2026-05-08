package s3client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tulvar/s3up/internal/list"
)

type Lister struct {
	client *s3.Client
}

func (l *Lister) List(ctx context.Context, input list.ListInput) ([]list.Entry, error) {
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(input.Bucket),
		Prefix: aws.String(input.Prefix),
	}
	if !input.Recursive {
		params.Delimiter = aws.String("/")
	}

	var entries []list.Entry
	paginator := s3.NewListObjectsV2Paginator(l.client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, prefix := range page.CommonPrefixes {
			if prefix.Prefix != nil {
				entries = append(entries, list.Entry{
					Prefix: *prefix.Prefix,
					IsDir:  true,
				})
			}
		}
		for _, object := range page.Contents {
			entry := list.Entry{
				Key:  aws.ToString(object.Key),
				Size: aws.ToInt64(object.Size),
				ETag: aws.ToString(object.ETag),
			}
			if object.LastModified != nil {
				entry.LastModified = *object.LastModified
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
