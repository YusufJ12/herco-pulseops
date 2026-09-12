.PHONY: help build test run docker-build docker-run docker-compose-up docker-compose-down clean

help:
	@echo "Perintah Development PulseOps:"
	@echo "  make test               - Menjalankan seluruh pengujian unit dan integrasi otomatis"
	@echo "  make docker-build       - Membangun image Docker production (pulseops:latest)"
	@echo "  make docker-compose-up  - Membangun dan menjalankan seluruh stack (PulseOps + Prometheus + Grafana)"
	@echo "  make docker-compose-down- Menghentikan seluruh container docker-compose"
	@echo "  make clean              - Membersihkan container dan volume persistent"

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
