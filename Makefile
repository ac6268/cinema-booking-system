.PHONY: run migrate-up migrate-down test

run:
	go run ./cmd/api

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

test:
	go test ./...