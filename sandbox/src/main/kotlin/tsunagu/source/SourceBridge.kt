package tsunagu.source

import kotlin.coroutines.intrinsics.COROUTINE_SUSPENDED
import kotlinx.coroutines.suspendCancellableCoroutine
import okhttp3.HttpUrl.Companion.toHttpUrl
import tsunagu.loader.LoadedExtension
import kotlin.coroutines.Continuation
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

class ExtensionBridgeException(message: String) : Exception(message)

class SourceBridge(private val extension: LoadedExtension) : KeiSource {
    private val target = extension.instance
    private val targetClass = target.javaClass

    override val name: String get() = readProperty("getName")
    override val lang: String get() = readProperty("getLang")
    override val id: Long get() = readProperty("getId")
    override val baseUrl: String get() = readProperty("getBaseUrl")

    override suspend fun getPopularManga(page: Int): List<SManga> {
        val result = callSuspend("getPopularManga", listOf(Int::class.java to page))
        return mapMangaList(extractMangas(result))
    }

    override suspend fun getSearchManga(page: Int, query: String, filters: FilterList): List<SManga> {
        val nativeFilters = nativeFilterList()
        val result = callSuspend(
            "getSearchManga",
            listOf(Int::class.java to page, String::class.java to query, nativeFilters.javaClass to nativeFilters),
        )
        return mapMangaList(extractMangas(result))
    }

    override suspend fun getMangaDetails(manga: SManga): SManga {
        val nativeManga = toNativeManga(manga)
        val result = callSuspend("getMangaDetails", listOf(nativeManga.javaClass.interfaces.first() to nativeManga))
        return toSManga(result!!)
    }

    override suspend fun getChapterList(manga: SManga): List<SChapter> {
        val nativeManga = toNativeManga(manga)
        val result = callSuspend("getChapterList", listOf(nativeManga.javaClass.interfaces.first() to nativeManga))
        return mapChapterList(result)
    }

    override suspend fun getPageList(chapter: SChapter): List<Page> {
        val nativeChapter = toNativeChapter(chapter)
        val result = callSuspend("getPageList", listOf(nativeChapter.javaClass.interfaces.first() to nativeChapter))
        return mapPageList(result)
    }

    override suspend fun getMangaByUrl(url: String): SManga? {
        if (!extension.isLegacy) {
            val nativeUrl = url.toHttpUrl()
            val result = callSuspend("getMangaByUrl", listOf(nativeUrl.javaClass to nativeUrl))
            if (result != null) return toSManga(result)
        }
        val nativeManga = toNativeManga(SManga(url = url, title = ""))
        val result = callSuspend("getMangaDetails", listOf(nativeManga.javaClass.interfaces.first() to nativeManga))
        return result?.let { toSManga(it) }
    }

    private fun <T> readProperty(getterName: String): T {
        @Suppress("UNCHECKED_CAST")
        return targetClass.getMethod(getterName).invoke(target) as T
    }

    private suspend fun callSuspend(methodName: String, args: List<Pair<Class<*>, Any?>>): Any? {
        val expectedParamCount = args.size + 1
        val method = targetClass.methods.firstOrNull { candidate ->
            candidate.name == methodName &&
                candidate.parameterCount == expectedParamCount &&
                args.indices.all { i -> candidate.parameterTypes[i].isAssignableFrom(args[i].first) }
        } ?: targetClass.methods.first { it.name == methodName && it.parameterCount == expectedParamCount }

        return suspendCancellableCoroutine { continuation ->
            val completion = object : Continuation<Any?> {
                override val context = continuation.context
                override fun resumeWith(result: Result<Any?>) {
                    result.fold(
                        onSuccess = { continuation.resume(it) },
                        onFailure = { continuation.resumeWithException(it) },
                    )
                }
            }
            val invokeArgs = args.map { it.second }.toTypedArray() + completion
            val invocationResult = method.invoke(target, *invokeArgs)
            if (invocationResult !== COROUTINE_SUSPENDED) {
                continuation.resume(invocationResult)
            }
        }
    }

    private fun nativeFilterList(): Any {
        val method = targetClass.methods.first { it.name == "getFilterList" && it.parameterCount == 0 }
        return method.invoke(target)
            ?: throw ExtensionBridgeException("getFilterList() on $targetClass returned null")
    }

    private fun extractMangas(mangasPage: Any?): List<Any> {
        if (mangasPage == null) return emptyList()
        @Suppress("UNCHECKED_CAST")
        return mangasPage.javaClass.getMethod("getMangas").invoke(mangasPage) as List<Any>
    }

    private fun toNativeManga(manga: SManga): Any {
        val mangaClass = Class.forName("eu.kanade.tachiyomi.source.model.SManga")
        val companion = mangaClass.getField("Companion").get(null)
        val created = companion.javaClass.getMethod("create").invoke(companion)
        setField(created, "url", manga.url)
        setField(created, "title", manga.title)
        manga.thumbnailUrl?.let { setField(created, "thumbnail_url", it) }
        return created
    }

    private fun toNativeChapter(chapter: SChapter): Any {
        val chapterClass = Class.forName("eu.kanade.tachiyomi.source.model.SChapter")
        val companion = chapterClass.getField("Companion").get(null)
        val created = companion.javaClass.getMethod("create").invoke(companion)
        setField(created, "url", chapter.url)
        setField(created, "name", chapter.name)
        return created
    }

    private fun toSManga(native: Any): SManga {
        val clazz = native.javaClass
        return SManga(
            url = getField<String>(clazz, native, "url"),
            title = getField<String>(clazz, native, "title"),
            artist = getFieldOrNull<String>(clazz, native, "artist"),
            author = getFieldOrNull<String>(clazz, native, "author"),
            description = getFieldOrNull<String>(clazz, native, "description"),
            genre = getFieldOrNull<String>(clazz, native, "genre"),
            thumbnailUrl = getFieldOrNull<String>(clazz, native, "thumbnail_url"),
        )
    }

    private fun toSChapter(native: Any): SChapter {
        val clazz = native.javaClass
        return SChapter(
            url = getField<String>(clazz, native, "url"),
            name = getField<String>(clazz, native, "name"),
            dateUpload = getFieldOrNull<Long>(clazz, native, "date_upload") ?: 0,
            chapterNumber = getFieldOrNull<Float>(clazz, native, "chapter_number") ?: -1f,
            scanlator = getFieldOrNull<String>(clazz, native, "scanlator"),
        )
    }

    private fun toPage(native: Any, index: Int): Page {
        val clazz = native.javaClass
        return Page(
            index = index,
            url = getFieldOrNull<String>(clazz, native, "url") ?: "",
            imageUrl = getFieldOrNull<String>(clazz, native, "imageUrl"),
        )
    }

    private fun mapMangaList(items: List<Any>): List<SManga> = items.map { toSManga(it) }

    private fun mapChapterList(result: Any?): List<SChapter> = extractList(result).map { toSChapter(it) }

    private fun mapPageList(result: Any?): List<Page> = extractList(result).mapIndexed { index, item -> toPage(item, index) }

    @Suppress("UNCHECKED_CAST")
    private fun extractList(result: Any?): List<Any> = when (result) {
        null -> emptyList()
        is List<*> -> result.filterNotNull()
        else -> emptyList()
    }

    private fun setField(instance: Any, fieldName: String, value: Any) {
        val field = findField(instance.javaClass, fieldName) ?: return
        field.isAccessible = true
        field.set(instance, value)
    }

    private fun <T> getField(clazz: Class<*>, instance: Any, fieldName: String): T {
        val field = findField(clazz, fieldName) ?: throw NoSuchFieldException(fieldName)
        field.isAccessible = true
        @Suppress("UNCHECKED_CAST")
        return field.get(instance) as T
    }

    private fun <T> getFieldOrNull(clazz: Class<*>, instance: Any, fieldName: String): T? {
        val field = findField(clazz, fieldName) ?: return null
        field.isAccessible = true
        @Suppress("UNCHECKED_CAST")
        return field.get(instance) as T?
    }

    private fun findField(clazz: Class<*>, fieldName: String): java.lang.reflect.Field? {
        var current: Class<*>? = clazz
        while (current != null) {
            runCatching { return current.getDeclaredField(fieldName) }
            current = current.superclass
        }
        return null
    }
}