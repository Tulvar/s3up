.PHONY: build integration s3-down s3-up test vulncheck

test:
	go test ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...

build:
	go build ./cmd/s3up

s3-up:
	docker compose up -d --wait --wait-timeout 45 rustfs

s3-down:
	docker compose down -v

integration: s3-up
	go test ./... -tags=integration
