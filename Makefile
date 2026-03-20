.PHONY: run build migrate-eazibiz migrate-mkwanja migrate-kanisa migrate-all sqlc test lint

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

migrate-eazibiz:
	goose -dir internal/db/migrations/consumer postgres "$(CONSUMER_EAZIBIZ_DATABASE_URL)" up

migrate-mkwanja:
	goose -dir internal/db/migrations/consumer postgres "$(CONSUMER_MKWANJA_DATABASE_URL)" up

migrate-kanisa:
	goose -dir internal/db/migrations/consumer postgres "$(CONSUMER_KANISA_DATABASE_URL)" up

migrate-all: migrate-eazibiz migrate-mkwanja migrate-kanisa

sqlc:
	sqlc generate

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...
