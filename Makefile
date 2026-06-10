.PHONY: run test build setup dev docker-up docker-down docker-api docker-certs lint migrate-up templ seed e2e e2e-go e2e-playwright-install

E2E_BASE_URL ?= https://127.0.0.1:8081
E2E_TAGS ?= e2e,playwright

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
	BASE_URL=$(E2E_BASE_URL) ./scripts/smoke_test.sh
	$(MAKE) e2e-go

e2e-playwright-install:
	@chmod +x scripts/install-playwright.sh
	@./scripts/install-playwright.sh

e2e-go:
	@command -v go >/dev/null 2>&1 && \
		$(MAKE) e2e-go-local || $(MAKE) e2e-go-docker

e2e-go-local: e2e-playwright-install
	BASE_URL=$(E2E_BASE_URL) go test -tags="$(E2E_TAGS)" ./test/e2e/... -count=1 -v

e2e-go-docker:
	@network=$$(docker inspect -f '{{range $$k, $$v := .NetworkSettings.Networks}}{{$$k}}{{end}}' homecoin-nginx 2>/dev/null); \
	if [ -z "$$network" ]; then \
		echo "homecoin-nginx not running — start stack with: docker compose up -d --wait"; \
		exit 1; \
	fi; \
	echo "Docker fallback: HTTP E2E only (Playwright needs local Go + Chromium)"; \
	docker run --rm --network "$$network" \
		-v "$(CURDIR):/app" -w /app \
		-e BASE_URL=https://nginx:443 \
		golang:1.25-alpine \
		sh -c "go test -tags=e2e ./test/e2e/... -count=1 -v"

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

