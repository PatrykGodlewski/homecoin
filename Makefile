.PHONY: run test build setup dev docker-up docker-down docker-api docker-certs lint templ seed migrations migrate e2e e2e-go e2e-playwright-install

E2E_BASE_URL ?= https://127.0.0.1:8081
E2E_TAGS ?= e2e,playwright

run: templ
	go run ./cmd/api

migrations:
	@chmod +x scripts/dev/sync-migrations.sh
	@./scripts/dev/sync-migrations.sh

migrate: migrations
	go run ./cmd/migrate

build: templ migrations
	go build -o bin/homecoin-api ./cmd/api
	go build -o bin/homecoin-worker ./cmd/worker
	go build -o bin/homecoin-migrate ./cmd/migrate

worker:
	go run ./cmd/worker

templ:
	@command -v templ >/dev/null 2>&1 || go install github.com/a-h/templ/cmd/templ@latest
	templ generate ./internal/ui/views/...

seed:
	@command -v go >/dev/null 2>&1 && go run ./cmd/seed || ./scripts/dev/seed_db.sh

test:
	go test ./... -count=1

e2e: docker-certs
	docker compose up -d --build --wait
	BASE_URL=$(E2E_BASE_URL) ./scripts/ci/smoke_test.sh
	$(MAKE) e2e-go

e2e-playwright-install:
	@chmod +x scripts/ci/install-playwright.sh
	@./scripts/ci/install-playwright.sh

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
	@chmod +x deploy/docker/nginx/generate-certs.sh
	@test -f deploy/docker/nginx/certs/tls.crt || ./deploy/docker/nginx/generate-certs.sh

lint:
	go vet ./...

setup:
	@test -f .env || cp .env.example .env
	@echo "Created .env from .env.example (edit JWT_SECRET for production)"

dev: setup docker-up
	@echo "Waiting for PostgreSQL..."
	@sleep 3
	$(MAKE) migrate
	$(MAKE) run

docker-up:
	docker compose up -d postgres

docker-down:
	docker compose down

docker-api: docker-certs
	docker compose up --build

