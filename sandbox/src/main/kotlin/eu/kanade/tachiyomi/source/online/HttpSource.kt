package eu.kanade.tachiyomi.source.online

import eu.kanade.tachiyomi.source.CatalogueSource
import eu.kanade.tachiyomi.source.model.FilterList
import eu.kanade.tachiyomi.source.model.MangasPage
import eu.kanade.tachiyomi.source.model.Page
import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga
import okhttp3.Headers
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import rx.Observable
import java.net.URI
import java.net.URISyntaxException

fun GET(url: String, headers: Headers = Headers.Builder().build()): Request =
    Request.Builder().url(url).headers(headers).build()

fun Observable<Response>.asObservableSuccess(): Observable<Response> {
    return this.map { response ->
        if (!response.isSuccessful) {
            response.close()
            throw java.io.IOException("HTTP error ${response.code}")
        }
        response
    }
}

abstract class HttpSource : CatalogueSource {
    abstract val baseUrl: String
    open val client: OkHttpClient = OkHttpClient()
    open val headers: Headers = Headers.Builder().build()
    override val supportsLatest: Boolean = false

    override val id: Long by lazy {
        val key = "${name.lowercase()}/$lang/$versionId"
        val bytes = java.security.MessageDigest.getInstance("MD5").digest(key.toByteArray())
        (0..7).map { bytes[it].toLong() and 0xff shl 8 * (7 - it) }.reduce(Long::or) and Long.MAX_VALUE
    }
    open val versionId: Int = 1

    protected open fun headersBuilder(): Headers.Builder = headers.newBuilder()

    private fun callObservable(request: Request): Observable<Response> {
        return Observable.defer {
            val response = client.newCall(request).execute()
            Observable.just(response)
        }
    }

    open fun popularMangaRequest(page: Int): Request = GET(baseUrl, headers)
    abstract fun popularMangaParse(response: Response): MangasPage
    open fun fetchPopularManga(page: Int): Observable<MangasPage> =
        callObservable(popularMangaRequest(page)).asObservableSuccess().map { response -> popularMangaParse(response) }

    open fun searchMangaRequest(page: Int, query: String, filters: FilterList): Request = GET(baseUrl, headers)
    abstract fun searchMangaParse(response: Response): MangasPage
    open fun fetchSearchManga(page: Int, query: String, filters: FilterList): Observable<MangasPage> =
        callObservable(searchMangaRequest(page, query, filters)).asObservableSuccess().map { response -> searchMangaParse(response) }

    open fun latestUpdatesRequest(page: Int): Request = GET(baseUrl, headers)
    abstract fun latestUpdatesParse(response: Response): MangasPage
    open fun fetchLatestUpdates(page: Int): Observable<MangasPage> =
        callObservable(latestUpdatesRequest(page)).asObservableSuccess().map { response -> latestUpdatesParse(response) }

    open fun mangaDetailsRequest(manga: SManga): Request = GET(baseUrl + manga.url, headers)
    abstract fun mangaDetailsParse(response: Response): SManga
    open fun fetchMangaDetails(manga: SManga): Observable<SManga> =
        callObservable(mangaDetailsRequest(manga)).asObservableSuccess().map { response -> mangaDetailsParse(response) }

    open fun chapterListRequest(manga: SManga): Request = GET(baseUrl + manga.url, headers)
    abstract fun chapterListParse(response: Response): List<SChapter>
    open fun fetchChapterList(manga: SManga): Observable<List<SChapter>> =
        callObservable(chapterListRequest(manga)).asObservableSuccess().map { response -> chapterListParse(response) }

    open fun pageListRequest(chapter: SChapter): Request = GET(baseUrl + chapter.url, headers)
    abstract fun pageListParse(response: Response): List<Page>
    open fun fetchPageList(chapter: SChapter): Observable<List<Page>> =
        callObservable(pageListRequest(chapter)).asObservableSuccess().map { response -> pageListParse(response) }

    open fun imageUrlParse(response: Response): String = throw NotImplementedError("imageUrlParse not implemented")
    open fun imageRequest(page: Page): Request = GET(page.imageUrl!!, headers)
    open fun fetchImageUrl(page: Page): Observable<String> =
        callObservable(imageRequest(page)).asObservableSuccess().map { response -> imageUrlParse(response) }

    override suspend fun getPopularManga(page: Int): MangasPage = fetchPopularManga(page).toBlocking().single()
    override suspend fun getSearchManga(page: Int, query: String, filters: FilterList): MangasPage = fetchSearchManga(page, query, filters).toBlocking().single()
    override suspend fun getLatestUpdates(page: Int): MangasPage = fetchLatestUpdates(page).toBlocking().single()
    override suspend fun getMangaDetails(manga: SManga): SManga = fetchMangaDetails(manga).toBlocking().single()
    override suspend fun getChapterList(manga: SManga): List<SChapter> = fetchChapterList(manga).toBlocking().single()
    open suspend fun getPageList(chapter: SChapter): List<Page> = fetchPageList(chapter).toBlocking().single()

    override fun getFilterList(): FilterList = FilterList()

    open fun getMangaUrl(manga: SManga): String = baseUrl + manga.url
    open fun getChapterUrl(chapter: SChapter): String = baseUrl + chapter.url

    fun SChapter.setUrlWithoutDomain(url: String) {
        this.url = getUrlWithoutDomain(url)
    }

    fun SManga.setUrlWithoutDomain(url: String) {
        this.url = getUrlWithoutDomain(url)
    }

    private fun getUrlWithoutDomain(orig: String): String =
        try {
            val uri = URI(orig.replace(" ", "%20"))
            var out = uri.path
            if (uri.query != null) {
                out += "?" + uri.query
            }
            if (uri.fragment != null) {
                out += "#" + uri.fragment
            }
            out
        } catch (_: URISyntaxException) {
            orig
        }
}
