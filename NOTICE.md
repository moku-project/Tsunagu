# Third-Party Notices

Tsunagu itself is licensed under the Apache License 2.0 (see `LICENSE`). It
also builds on the Tachiyomi/Mihon extension ecosystem and is
architecturally inspired by Suwayomi. This file lists what's reused or
adapted, from where, and under what license. Full third-party license texts
are in `THIRD_PARTY_LICENSES/`.

## Mihon (Tachiyomi) — Apache License 2.0

<https://github.com/mihonapp/mihon>

Tachiyomi's manga-extension API (`Source`, `HttpSource`, `SManga`, `SChapter`,
`Filter`, the `network.*` OkHttp helpers, etc.) is what third-party manga
extensions are written against. Tsunagu's sandbox reimplements this API
surface, adapted from Mihon (Tachiyomi's actively maintained successor after
the original project's closure), so those extensions run unmodified inside
our own JVM sandbox instead of Tachiyomi's Android app.

Adapted under `sandbox/src/main/kotlin/eu/kanade/tachiyomi/source/`,
`.../network/`, and `.../util/`.

Tsunagu's Mihon-format backup import/export (`.tachibk`) also mirrors Mihon's
`backup.proto` schema field-for-field, so backups are interchangeable with
Mihon/Tachiyomi. See `backend/internal/backup/mihonpb/backup.proto`.

## Aniyomi — Apache License 2.0

<https://github.com/aniyomiorg/aniyomi>

Aniyomi is Mihon's anime-focused fork, and defines the parallel
`AnimeSource`/`AnimeHttpSource`/`SAnime`/`SEpisode`/`Video` API that anime
extensions are written against. Adapted under
`sandbox/src/main/kotlin/eu/kanade/tachiyomi/animesource/`.

## Tsundoku extensions-lib — Apache License 2.0

<https://github.com/tsundoku-otaku/extensions-lib>

Tsundoku's fork of the Mihon extension API adds the interfaces its novel
extensions (e.g. the NovelSourcery repo) are compiled against. Adapted as
`SourceTracker` and `RateLimited` under
`sandbox/src/main/kotlin/eu/kanade/tachiyomi/source/`.

## Suwayomi — Mozilla Public License 2.0 (architectural reference, no copied code)

<https://github.com/Suwayomi/Suwayomi-Server>

Suwayomi pioneered running the Tachiyomi extension ecosystem inside a
headless JVM server and exposing it to non-Android clients — the same shape
as Tsunagu's own sandbox/backend split. No Suwayomi source is copied or
adapted here; this credits the architectural approach, licensed separately
under MPL-2.0 for anyone consulting the original.

## License compliance

Per Apache License 2.0 §4, adapted files retain a notice pointing back to
their origin, and a copy of the license is included in
`THIRD_PARTY_LICENSES/Apache-2.0.txt`. The MPL-2.0 text for Suwayomi is
included at `THIRD_PARTY_LICENSES/MPL-2.0.txt` for reference, even though no
MPL-covered code is included in this repository.
