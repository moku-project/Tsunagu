package tsunagu.source

data class SManga(
    val url: String,
    val title: String,
    val artist: String? = null,
    val author: String? = null,
    val description: String? = null,
    val genre: String? = null,
    val status: Int = 0,
    val thumbnailUrl: String? = null,
)

data class SChapter(
    val url: String,
    val name: String,
    val dateUpload: Long = 0,
    val chapterNumber: Float = -1f,
    val scanlator: String? = null,
)

data class Page(
    val index: Int,
    val url: String = "",
    val imageUrl: String? = null,
)

data class FilterList(val filters: List<Any> = emptyList())

interface KeiSource {
    val name: String
    val lang: String
    val id: Long
    val baseUrl: String

    suspend fun getPopularManga(page: Int): List<SManga>
    suspend fun getSearchManga(page: Int, query: String, filters: FilterList): List<SManga>
    suspend fun getMangaDetails(manga: SManga): SManga
    suspend fun getChapterList(manga: SManga): List<SChapter>
    suspend fun getPageList(chapter: SChapter): List<Page>
    suspend fun getMangaByUrl(url: String): SManga?
}
