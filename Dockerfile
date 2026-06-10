FROM golang:1.25-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/a-h/templ/cmd/templ@latest

COPY . .
RUN cp migrations/*.sql internal/infrastructure/postgres/migrations/
RUN templ generate ./internal/ui/views/...
RUN CGO_ENABLED=0 GOOS=linux go build -o /homecoin-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /homecoin-worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /homecoin-migrate ./cmd/migrate

FROM alpine:3.20 AS api

RUN apk add --no-cache ca-certificates wget
WORKDIR /app

COPY --from=builder /homecoin-api /app/homecoin
COPY .env.example .env

EXPOSE 8080
CMD ["/app/homecoin"]

FROM alpine:3.20 AS worker

RUN apk add --no-cache ca-certificates wget
WORKDIR /app

COPY --from=builder /homecoin-worker /app/homecoin-worker

EXPOSE 8080
CMD ["/app/homecoin-worker"]

FROM alpine:3.20 AS migrate

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /homecoin-migrate /app/homecoin-migrate

CMD ["/app/homecoin-migrate"]
