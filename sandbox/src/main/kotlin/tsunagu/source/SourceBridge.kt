package tsunagu.source

import eu.kanade.tachiyomi.source.CatalogueSource
import eu.kanade.tachiyomi.source.model.FilterList
import eu.kanade.tachiyomi.source.model.MangasPage
import eu.kanade.tachiyomi.source.model.Page
import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga
import eu.kanade.tachiyomi.source.model.SMangaUpdate
import eu.kanade.tachiyomi.source.online.HttpSource
import kotlinx.coroutines.async
import kotlinx.coroutines.supervisorScope
import tsunagu.util.awaitSingle

suspend fun CatalogueSource.getPopularManga(page: Int): MangasPage =
    fetchPopularManga(page).awaitSingle()

suspend fun CatalogueSource.getLatestUpdates(page: Int): MangasPage =
    fetchLatestUpdates(page).awaitSingle()

suspend fun CatalogueSource.getSearchManga(page: Int, query: String, filters: FilterList): MangasPage =
    fetchSearchManga(page, query, filters).awaitSingle()

suspend fun HttpSource.getMangaUpdate(
    manga: SManga,
    chapters: List<SChapter>,
    fetchDetails: Boolean,
    fetchChapters: Boolean,
): SMangaUpdate = supervisorScope {
    val asyncManga = if (fetchDetails) async { fetchMangaDetails(manga).awaitSingle() } else null
    val asyncChapters = if (fetchChapters) async { fetchChapterList(manga).awaitSingle() } else null
    SMangaUpdate(asyncManga?.await() ?: manga, asyncChapters?.await() ?: chapters)
}

suspend fun HttpSource.getPageList(chapter: SChapter): List<Page> =
    fetchPageList(chapter).awaitSingle()
