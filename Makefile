.PHONY: build integration minio-down minio-up test

test:
	go test ./...

build:
	go build ./cmd/s3up

minio-up:
	docker compose up -d minio

minio-down:
	docker compose down -v

integration: minio-up
	go test ./... -tags=integration
