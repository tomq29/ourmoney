.PHONY: up down logs build test lint run

up:
	docker compose up --build -d
	docker compose ps

down:
	docker compose down -v

logs:
	docker compose logs -f bot

build:
	go build ./...

test:
	go test ./... -race -count=1

lint:
	test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...

run:
	DATABASE_URL=postgres://ourmoney:ourmoney@localhost:$${POSTGRES_PORT:-5432}/ourmoney?sslmode=disable go run ./cmd/bot
