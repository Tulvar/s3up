.PHONY: build integration minio-down minio-up test vulncheck

test:
	go test ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...

build:
	go build ./cmd/s3up

minio-up:
	docker compose up -d minio

minio-down:
	docker compose down -v

integration: minio-up
	go test ./... -tags=integration
