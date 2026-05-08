# s3up

Small CLI tool for uploading files and directories to S3.

## Usage

```sh
s3up upload ./file.txt s3://bucket/path/file.txt
s3up upload ./dist s3://bucket/site/ --recursive
```

Useful flags:

```sh
--profile       AWS shared config profile
--region        AWS region
--endpoint-url  custom S3-compatible endpoint
--path-style    use path-style addressing
--recursive     upload directories recursively
--dry-run       print planned uploads without sending objects
--no-progress   disable upload progress output
--content-type  content type for uploaded objects
--metadata      object metadata as key=value, can be repeated
--storage-class S3 storage class
--acl           S3 canned ACL
--concurrency   multipart upload concurrency
--part-size     multipart part size, for example 8MiB
```

Example for a local S3-compatible service:

```sh
s3up upload ./file.txt s3://bucket/file.txt \
  --endpoint-url http://localhost:9000 \
  --region us-east-1 \
  --path-style
```

Example with object options:

```sh
s3up upload ./dist s3://bucket/site/ \
  --recursive \
  --content-type text/html \
  --metadata app=s3up \
  --metadata env=prod \
  --storage-class STANDARD_IA \
  --acl bucket-owner-full-control \
  --concurrency 8 \
  --part-size 16MiB
```

## Development

```sh
go test ./...
go build ./cmd/s3up
```

Or use Make:

```sh
make test
make build
```

## Integration Tests

Integration tests use MinIO as an S3-compatible endpoint.

```sh
make integration
```

To stop and remove the MinIO volume:

```sh
make minio-down
```
