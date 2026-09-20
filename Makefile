.PHONY: build test run lint docker-build up down logs

build:
	go build -o bin/usdt-rate-server ./cmd/server

test:
	go test ./...

run: build
	./bin/usdt-rate-server

lint:
	golangci-lint run ./...

docker-build:
	docker build -t usdt-rate-service:local .

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f server
