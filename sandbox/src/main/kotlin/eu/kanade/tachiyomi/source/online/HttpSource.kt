package eu.kanade.tachiyomi.source.online

import eu.kanade.tachiyomi.network.NetworkHelper
import eu.kanade.tachiyomi.network.asObservableSuccess
import org.koin.core.context.GlobalContext
import eu.kanade.tachiyomi.source.CatalogueSource
import eu.kanade.tachiyomi.source.model.*
import kotlinx.coroutines.async
import kotlinx.coroutines.supervisorScope
import okhttp3.Headers
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import rx.Observable
import tsunagu.util.awaitSingle
import java.security.MessageDigest

@Suppress("unused", "unused_parameter")
abstract class HttpSource : CatalogueSource {

    protected val network: NetworkHelper get() = GlobalContext.get().get()

    abstract val baseUrl: String

    open val versionId: Int = 1

    override val id: Long by lazy {
        val key = "${name.lowercase()}/$lang/$versionId"
        val bytes = MessageDigest.getInstance("MD5").digest(key.toByteArray())
        (0..7).map { bytes[it].toLong() and 0xff shl 8 * (7 - it) }.reduce(Long::or) and Long.MAX_VALUE
    }

    val headers: Headers by lazy { headersBuilder().build() }

    open val client: OkHttpClient get() = network.client

    protected open fun headersBuilder(): Headers.Builder = Headers.Builder().apply {
        add("User-Agent", network.defaultUserAgentProvider())
    }

    override fun toString(): String = "$name (${lang.uppercase()})"

    @Deprecated("Use the suspend API instead", ReplaceWith("getPopularManga"))
    override fun fetchPopularManga(page: Int): Observable<MangasPage> {
        return client.newCall(popularMangaRequest(page))
            .asObservableSuccess()
            .map { popularMangaParse(it) }
    }

    protected open fun popularMangaRequest(page: Int): Request = throw UnsupportedOperationException()

    protected open fun popularMangaParse(response: Response): MangasPage = throw UnsupportedOperationException()

    @Deprecated("Use the suspend API instead", ReplaceWith("getSearchManga"))
    override fun fetchSearchManga(page: Int, query: String, filters: FilterList): Observable<MangasPage> {
        return client.newCall(searchMangaRequest(page, query, filters))
            .asObservableSuccess()
            .map { searchMangaParse(it) }
    }

    protected open fun searchMangaRequest(page: Int, query: String, filters: FilterList): Request = throw UnsupportedOperationException()

    protected open fun searchMangaParse(response: Response): MangasPage = throw UnsupportedOperationException()

    @Deprecated("Use the suspend API instead", ReplaceWith("getLatestUpdates"))
    override fun fetchLatestUpdates(page: Int): Observable<MangasPage> {
        return client.newCall(latestUpdatesRequest(page))
            .asObservableSuccess()
            .map { latestUpdatesParse(it) }
    }

    protected open fun latestUpdatesRequest(page: Int): Request = throw UnsupportedOperationException()

    protected open fun latestUpdatesParse(response: Response): MangasPage = throw UnsupportedOperationException()

    @Deprecated("Use the combined suspend API instead", ReplaceWith("getMangaUpdate"))
    override fun fetchMangaDetails(manga: SManga): Observable<SManga> {
        return client.newCall(mangaDetailsRequest(manga))
            .asObservableSuccess()
            .map { mangaDetailsParse(it).apply { initialized = true } }
    }

    open fun mangaDetailsRequest(manga: SManga): Request = GET(baseUrl + manga.url, headers)

    protected open fun mangaDetailsParse(response: Response): SManga = throw UnsupportedOperationException()

    override val supportsRelatedMangas: Boolean get() = true

    override suspend fun fetchRelatedMangaList(manga: SManga): List<SManga> {
        return client.newCall(relatedMangaListRequest(manga))
            .asObservableSuccess()
            .map { relatedMangaListParse(it) }
            .awaitSingle()
    }

    protected open fun relatedMangaListRequest(manga: SManga): Request = GET(baseUrl + manga.url, headers)

    protected open fun relatedMangaListParse(response: Response): List<SManga> = throw UnsupportedOperationException()

    @Deprecated("Use the combined suspend API instead", ReplaceWith("getMangaUpdate"))
    override fun fetchChapterList(manga: SManga): Observable<List<SChapter>> {
        return client.newCall(chapterListRequest(manga))
            .asObservableSuccess()
            .map { chapterListParse(it) }
    }

    protected open fun chapterListRequest(manga: SManga): Request = GET(baseUrl + manga.url, headers)

    protected open fun chapterListParse(response: Response): List<SChapter> = throw UnsupportedOperationException()

    override suspend fun getMangaUpdate(
        manga: SManga,
        chapters: List<SChapter>,
        fetchDetails: Boolean,
        fetchChapters: Boolean,
    ): SMangaUpdate = supervisorScope {
        val asyncManga = if (fetchDetails) async { fetchMangaDetails(manga).awaitSingle() } else null
        val asyncChapters = if (fetchChapters) async { fetchChapterList(manga).awaitSingle() } else null
        SMangaUpdate(asyncManga?.await() ?: manga, asyncChapters?.await() ?: chapters)
    }

    @Deprecated("Use the suspend API instead", ReplaceWith("getPageList"))
    override fun fetchPageList(chapter: SChapter): Observable<List<Page>> {
        return client.newCall(pageListRequest(chapter))
            .asObservableSuccess()
            .map { pageListParse(it) }
    }

    override suspend fun getPageList(chapter: SChapter): List<Page> =
        fetchPageList(chapter).awaitSingle()

    protected open fun pageListRequest(chapter: SChapter): Request = GET(baseUrl + chapter.url, headers)

    protected open fun pageListParse(response: Response): List<Page> = throw UnsupportedOperationException()

    open fun fetchImageUrl(page: Page): Observable<String> {
        return client.newCall(imageUrlRequest(page))
            .asObservableSuccess()
            .map { imageUrlParse(it) }
    }

    open suspend fun getImageUrl(page: Page): String =
        fetchImageUrl(page).awaitSingle()

    protected open fun imageUrlRequest(page: Page): Request = GET(page.url, headers)

    protected open fun imageUrlParse(response: Response): String = throw UnsupportedOperationException()

    fun fetchImage(page: Page): Observable<Response> {
        return client.newCall(imageRequest(page)).asObservableSuccess()
    }

    protected open fun imageRequest(page: Page): Request = GET(page.imageUrl!!, headers)

    fun SChapter.setUrlWithoutDomain(url: String) {
        this.url = getUrlWithoutDomain(url)
    }

    fun SManga.setUrlWithoutDomain(url: String) {
        this.url = getUrlWithoutDomain(url)
    }

    private fun getUrlWithoutDomain(orig: String): String {
        return try {
            val url = if (orig.startsWith("http")) orig.toHttpUrl() else "$baseUrl$orig".toHttpUrl()
            var out = url.encodedPath
            if (url.encodedQuery != null) out += "?${url.encodedQuery}"
            if (url.encodedFragment != null) out += "#${url.encodedFragment}"
            out
        } catch (_: Exception) {
            orig
        }
    }

    open fun getMangaUrl(manga: SManga): String {
        return try {
            mangaDetailsRequest(manga).url.toString()
        } catch (_: Exception) {
            baseUrl + manga.url
        }
    }

    open fun getChapterUrl(chapter: SChapter): String {
        return try {
            pageListRequest(chapter).url.toString()
        } catch (_: Exception) {
            baseUrl + chapter.url
        }
    }

    open fun prepareNewChapter(chapter: SChapter, manga: SManga) {}

    override fun getFilterList(): FilterList = FilterList()

    protected fun GET(url: String, headers: Headers = this.headers): Request =
        Request.Builder().url(url).headers(headers).build()
}