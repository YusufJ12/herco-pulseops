.PHONY: help build test run docker-build docker-run docker-compose-up docker-compose-down clean

help:
	@echo "PulseOps Development Commands:"
	@echo "  make test               - Run all automated unit and integration tests"
	@echo "  make docker-build       - Build the production Docker image"
	@echo "  make docker-compose-up  - Start services via docker-compose"
	@echo "  make docker-compose-down- Stop services via docker-compose"

test:
	docker run --rm -v "$$(pwd):/app" -w /app golang:1.23-alpine go test -v ./...

docker-build:
	docker build -t pulseops:latest .

docker-compose-up:
	docker compose up -d --build

docker-compose-down:
	docker compose down

clean:
	docker compose down -v
