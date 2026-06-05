.PHONY: run test migrate-up migrate-down migrate-create sqlc docker-up docker-down lint

run:
	go run ./cmd/api

test:
	go test ./... -count=1 -race

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

sqlc:
	sqlc generate

docker-up:
	docker compose up -d

docker-down:
	docker compose down

lint:
	go vet ./...
