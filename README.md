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

## Packaging

- **Arch** — [`packaging/tsunagu`](packaging/tsunagu/PKGBUILD) (latest tagged release) or [`packaging/tsunagu-git`](packaging/tsunagu-git/PKGBUILD) (latest commit), built with `makepkg -si` since neither is on the AUR yet. Either installs `/usr/bin/tsunagu` plus the sandbox jar it needs. [Moku](https://github.com/moku-project/Moku)'s default `moku`/`moku-git` PKGBUILDs depend on these to bundle the backend automatically.
- **Nix** — `nix build .#tsunagu` (see `nix/packages.nix`).

## Auth

Two independent, optional gates in front of the API:

- **Static API token** — set `api_token` in `tsunagu.toml`; clients send `Authorization: Bearer <token>` or `?token=`.
- **Password + sessions** — set a password (`setPassword` mutation, or from a connected client's settings). Login via `POST /api/auth/login` returns a 30-day HMAC-signed session token; `GET /api/auth/status` reports whether a password is required. Either credential is accepted independently.

Neither is required for local/loopback use — auth only matters once the server is reachable from outside your machine.

## Backups

- **Mihon/Tachiyomi-format export** (`.tachibk`) — `exportMihonBackup`/`importMihonBackup` mutations. Manga/novel library only; anime is never included, since the format has no concept of it.
- **SQLite snapshots** — `createDatabaseBackup`/`deleteDatabaseBackup` mutations, plus automatic scheduled snapshots via `backup_interval_hours`/`backup_retention_count` in `tsunagu.toml`.

## Moku Sync

Paired instances keep the library, reading progress, bookmarks, folders and reading history in step over the local network. No account, no cloud. Extensions and downloads stay per device; titles whose extension a device lacks are listed in `syncStatus.missingExtensions`.

- **Pairing** — each device needs a name first (`setSyncDeviceName`). One device shows a one-time code (`createSyncPairingCode`); the other previews what will merge (`previewSyncPeer`) and pairs (`connectSyncPeer`). Either side can start a sync after that.
- **Discovery** — instances announce `_moku._tcp` over mDNS and `syncNearby` lists them. Apps that can't open multicast sockets (iOS) browse and advertise natively via the embedded `SetSyncDiscovery`/`SyncAdvert`, and call `SyncWake` on returning to the foreground.
- **Port** — pairing and sync are served on `sync_addr` (default `:6008`), separate from the API. Allow TCP 6008 and UDP 5353 through firewalls, or set `sync_addr = ""` to turn it off. Traffic is plain HTTP, so use Tailscale or HTTPS to sync beyond your own network.
- **History** — `recordReadingSession` stores reading sessions; `readingSessions`, `readingStats` and `Media.resumePoint` read them back, synced like everything else.

## Acknowledgements

Tsunagu's sandbox reimplements the Tachiyomi/Mihon extension API so
third-party manga and anime extensions run unmodified in our own JVM
process, and its backup format is interchangeable with Mihon's. Its overall
shape — a headless JVM extension host exposed to a non-JVM backend — follows
the approach pioneered by Suwayomi.

- **[Mihon](https://github.com/mihonapp/mihon)** (Tachiyomi's successor) — manga extension API and backup format. Apache License 2.0.
- **[Aniyomi](https://github.com/aniyomiorg/aniyomi)** — anime extension API. Apache License 2.0.
- **[Suwayomi](https://github.com/Suwayomi/Suwayomi-Server)** — architectural reference for the sandbox/backend split; no code copied. Mozilla Public License 2.0.

See [`NOTICE.md`](NOTICE.md) for exactly which files are adapted from where,
and [`THIRD_PARTY_LICENSES/`](THIRD_PARTY_LICENSES/) for full license texts.

## License

Apache License 2.0 — see [`LICENSE`](LICENSE).
