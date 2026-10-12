// Adapted from Tsundoku extensions-lib (https://github.com/tsundoku-otaku/extensions-lib), licensed under Apache License 2.0.
// See /NOTICE.md and /THIRD_PARTY_LICENSES/Apache-2.0.txt for full attribution.

package eu.kanade.tachiyomi.source

import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga

/** Tsundoku novel sources implement this to sync read state back to the site; the sandbox never calls it. */
@Suppress("unused")
interface SourceTracker {
    val supportsChapterTracking: Boolean
        get() = true

    val supportsFavoritesTracking: Boolean
        get() = false

    suspend fun onChaptersRead(
        manga: SManga,
        changedChapters: List<SChapter>,
        allChapters: List<SChapter>,
        categories: List<String>,
    ) = Unit

    suspend fun onChaptersUnread(
        manga: SManga,
        changedChapters: List<SChapter>,
        allChapters: List<SChapter>,
        categories: List<String>,
    ) = Unit

    suspend fun onFavorited(manga: SManga, categories: List<String>) = Unit

    suspend fun onUnfavorited(manga: SManga, categories: List<String>) = Unit
}
