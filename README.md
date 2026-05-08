# s3up

Small CLI tool for uploading files and directories to S3.

## Usage

```sh
s3up upload ./file.txt s3://bucket/path/file.txt
s3up upload ./dist s3://bucket/site/ --recursive
s3up upload ./dist s3://bucket/site/ --recursive --exclude "*.map"
s3up upload ./dist s3://bucket/site/ --recursive --include "*.html"
s3up sync ./dist s3://bucket/site/
s3up sync ./dist s3://bucket/site/ --dry-run
s3up sync ./dist s3://bucket/site/ --checksum
s3up sync ./dist s3://bucket/site/ --exclude "*.map"
s3up sync ./dist s3://bucket/site/ --include "*.html"
s3up ls s3://bucket/site/
s3up ls s3://bucket/site/ --recursive
s3up ls s3://bucket/site/ --human
s3up ls s3://bucket/site/ --json
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
--include       include files by glob pattern, can be repeated
--exclude       exclude files by glob pattern, can be repeated
```

List flags:

```sh
--profile       AWS shared config profile
--region        AWS region
--endpoint-url  custom S3-compatible endpoint
--path-style    use path-style addressing
--recursive     list objects recursively
--human         print object sizes in human-readable units
--json          print entries as JSON
```

Sync flags:

```sh
--profile       AWS shared config profile
--region        AWS region
--endpoint-url  custom S3-compatible endpoint
--path-style    use path-style addressing
--dry-run       print sync actions without uploading objects
--checksum      compare same-size objects using local MD5 and remote ETag
--no-progress   disable sync progress output
--content-type  content type for uploaded objects
--metadata      object metadata as key=value, can be repeated
--storage-class S3 storage class
--acl           S3 canned ACL
--concurrency   multipart upload concurrency
--part-size     multipart part size, for example 8MiB
--include       include files by glob pattern, can be repeated
--exclude       exclude files by glob pattern, can be repeated
```

Sync uploads local files that are missing remotely or have a different size. It
does not delete remote objects.

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
