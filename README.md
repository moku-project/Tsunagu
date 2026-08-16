# Tsunagu

繋ぐ — "to connect."

Self-hosted backend for manga, light novels, and anime.

## Getting started

```sh
nix develop
cp .env.example .env
make dev
```

- GraphQL: `http://localhost:8080/graphql`
- Playground: `http://localhost:8080/playground`
- REST + OpenAPI: `http://localhost:8080/docs`
- Postman collection: `docs/tsunagu.postman_collection.json`

## Layout

- `backend/` — Go API server, DB, download queue, image/EPUB pipelines
- `sandbox/` — JVM extension loader/executor
- `proto/` — shared gRPC contract
