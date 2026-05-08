# s3up

Small CLI tool for uploading files and directories to S3.

## Usage

```sh
s3up upload ./file.txt s3://bucket/path/file.txt
s3up upload ./dist s3://bucket/site/ --recursive
s3up upload ./dist s3://bucket/site/ --recursive --exclude "*.map"
s3up upload ./dist s3://bucket/site/ --recursive --include "*.html"
s3up upload ./dist s3://bucket/site/ --recursive --workers 8
s3up sync ./dist s3://bucket/site/
s3up sync ./dist s3://bucket/site/ --dry-run
s3up sync ./dist s3://bucket/site/ --checksum
s3up sync ./dist s3://bucket/site/ --delete --yes
s3up sync ./dist s3://bucket/site/ --delete --dry-run
s3up sync ./dist s3://bucket/site/ --exclude "*.map"
s3up sync ./dist s3://bucket/site/ --include "*.html"
s3up sync ./dist s3://bucket/site/ --workers 8
s3up delete s3://bucket/site/old.js
s3up delete s3://bucket/site/ --recursive --dry-run
s3up delete s3://bucket/site/ --recursive --yes --exclude "*.map"
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
--workers       number of files to process in parallel
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

Delete flags:

```sh
--profile       AWS shared config profile
--region        AWS region
--endpoint-url  custom S3-compatible endpoint
--path-style    use path-style addressing
--recursive     delete objects recursively under the target prefix
--dry-run       print delete actions without deleting objects
--yes           confirm recursive delete
--no-progress   disable delete progress output
--workers       number of delete actions to process in parallel
--include       include files by glob pattern, can be repeated
--exclude       exclude files by glob pattern, can be repeated
```

Sync flags:

```sh
--profile       AWS shared config profile
--region        AWS region
--endpoint-url  custom S3-compatible endpoint
--path-style    use path-style addressing
--dry-run       print sync actions without uploading objects
--checksum      compare same-size objects using local MD5 and remote ETag
--delete        delete remote objects missing from the local plan
--yes           confirm destructive sync actions such as --delete
--no-progress   disable sync progress output
--content-type  content type for uploaded objects
--metadata      object metadata as key=value, can be repeated
--storage-class S3 storage class
--acl           S3 canned ACL
--concurrency   multipart upload concurrency
--workers       number of sync actions to process in parallel
--part-size     multipart part size, for example 8MiB
--include       include files by glob pattern, can be repeated
--exclude       exclude files by glob pattern, can be repeated
```

Sync uploads local files that are missing remotely or have a different size. By
default it does not delete remote objects. Pass `--delete --dry-run` to preview
removals, or `--delete --yes` to remove remote objects under the destination
prefix that are missing from the filtered local plan.

When progress output is enabled, upload and sync commands print a final summary
with uploaded, deleted, skipped, or planned action counts.

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
