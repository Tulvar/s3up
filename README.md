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
```

Example for a local S3-compatible service:

```sh
s3up upload ./file.txt s3://bucket/file.txt \
  --endpoint-url http://localhost:9000 \
  --region us-east-1 \
  --path-style
```

## Development

```sh
go test ./...
```
