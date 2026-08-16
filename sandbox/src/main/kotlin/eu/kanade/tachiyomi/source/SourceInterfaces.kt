// Hand-written subset of the legacy tachiyomi.source.Source/CatalogueSource
// interfaces (getMangaDetails/getChapterList style, not the newer
// getMangaUpdate combined API used by current Suwayomi). Matches what
// SourceBridge.kt reflects against. Do not replace with a verbatim copy
// of Suwayomi/eu.kanade.tachiyomi.source.Source.kt without also updating
// SourceBridge.kt to match the newer method signatures.

package eu.kanade.tachiyomi.source

import eu.kanade.tachiyomi.source.model.FilterList
import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga

interface Source {
    val id: Long
    val name: String

    suspend fun getMangaDetails(manga: SManga): SManga
    suspend fun getChapterList(manga: SManga): List<SChapter>
}

interface CatalogueSource : Source {
    val lang: String
    val supportsLatest: Boolean

    suspend fun getPopularManga(page: Int): eu.kanade.tachiyomi.source.model.MangasPage
    suspend fun getSearchManga(page: Int, query: String, filters: FilterList): eu.kanade.tachiyomi.source.model.MangasPage
    suspend fun getLatestUpdates(page: Int): eu.kanade.tachiyomi.source.model.MangasPage
    fun getFilterList(): FilterList
}
