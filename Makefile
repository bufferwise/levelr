.PHONY: dev build sqlc migrate-up migrate-down lint clean

include .env
export

dev:
	go run ./cmd/bot/.

build:
	go build -o bin/bot ./cmd/bot/.

sqlc:
	sqlc generate

migrate-up:
	migrate -path internal/db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DB_URL)" down 1

migrate-create:
	migrate create -ext sql -dir internal/db/migrations -seq $(name)

lint:
	golangci-lint run ./...

vet:
	go vet ./...

test:
	go test ./... -v -race

clean:
	rm -rf bin/

prune-old-weeks:
	psql "$(DB_URL)" -c "DELETE FROM weekly_messages WHERE week_start < NOW() - INTERVAL '90 days'; DELETE FROM weekly_voice WHERE week_start < NOW() - INTERVAL '90 days';"
