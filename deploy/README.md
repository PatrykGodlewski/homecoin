# Local deployment (Docker Compose)

TLS reverse proxy for local development and CI E2E tests.

```
deploy/docker/nginx/
├── nginx.conf           # HTTPS → api:8080
├── generate-certs.sh    # self-signed cert (dev only)
└── certs/               # gitignored — created by generate-certs.sh
```

```bash
make docker-certs    # or: ./deploy/docker/nginx/generate-certs.sh
docker compose up --build
```

Production uses Azure Container Apps ingress (HTTPS) — not this nginx stack.
