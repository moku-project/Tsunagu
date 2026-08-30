# Tsunagu

Self-hosted backend for manga, light novels, and anime.

```sh
nix develop
cd backend && go run ./cmd/server
```

GraphQL at `http://localhost:6007/api/graphql` (playground at `/api/graphql/playground`).

- `backend/` — Go API server, DB, download queue, media pipelines
- `sandbox/` — JVM extension loader/executor
- `proto/` — shared gRPC contract
