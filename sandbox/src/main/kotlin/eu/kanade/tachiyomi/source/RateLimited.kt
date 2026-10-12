// Adapted from Tsundoku extensions-lib (https://github.com/tsundoku-otaku/extensions-lib), licensed under Apache License 2.0.
// See /NOTICE.md and /THIRD_PARTY_LICENSES/Apache-2.0.txt for full attribution.

package eu.kanade.tachiyomi.source

/** Pacing hints Tsundoku uses for bulk downloads; sources still rate-limit their own clients. */
@Suppress("unused")
interface RateLimited {
    val minimumDelayMillis: Long

    val recommendedDelayMillis: Long
        get() = minimumDelayMillis

    val recommendedPermits: Int
        get() = 1
}
