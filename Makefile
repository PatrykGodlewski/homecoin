.PHONY: run test build setup dev docker-up docker-down docker-api docker-certs lint migrate-up templ seed e2e

run: templ
	go run ./cmd/api

build: templ
	go build -o bin/homecoin-api ./cmd/api
	go build -o bin/homecoin-worker ./cmd/worker

worker:
	go run ./cmd/worker

templ:
	@command -v templ >/dev/null 2>&1 || go install github.com/a-h/templ/cmd/templ@latest
	templ generate ./internal/ui/views/...

seed:
	@command -v go >/dev/null 2>&1 && go run ./cmd/seed || ./scripts/seed_db.sh

test:
	go test ./... -count=1

e2e: docker-certs
	docker compose up -d --build --wait
	BASE_URL=https://localhost:8081 ./scripts/smoke_test.sh
	go test -tags=e2e ./test/e2e/... -count=1 -v

docker-certs:
	@chmod +x deploy/nginx/generate-certs.sh
	@test -f deploy/nginx/certs/tls.crt || ./deploy/nginx/generate-certs.sh

lint:
	go vet ./...

setup:
	@test -f .env || cp .env.example .env
	@echo "Created .env from .env.example (edit JWT_SECRET for production)"

dev: setup docker-up
	@echo "Waiting for PostgreSQL..."
	@sleep 3
	$(MAKE) run

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down

docker-api: docker-certs
	docker compose up --build

