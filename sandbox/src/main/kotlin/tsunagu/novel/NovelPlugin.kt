package tsunagu.novel

import org.graalvm.polyglot.Context
import org.graalvm.polyglot.Value

data class NovelItem(val name: String, val path: String, val cover: String? = null)

data class ChapterItem(
    val name: String,
    val path: String,
    val chapterNumber: Double? = null,
    val releaseTime: String? = null,
    val page: String? = null,
)

data class SourceNovel(
    val name: String,
    val path: String,
    val cover: String? = null,
    val genres: String? = null,
    val summary: String? = null,
    val author: String? = null,
    val artist: String? = null,
    val status: String? = null,
    val chapters: List<ChapterItem> = emptyList(),
    val totalPages: Int? = null,
)

class NovelPluginException(message: String) : Exception(message)

class NovelPlugin(
    val id: String,
    val name: String,
    val site: String,
    val lang: String,
    val version: String,
    private val context: Context,
    private val pluginValue: Value,
) : AutoCloseable {

    private val callAsync: Value = context.eval(
        "js",
        """
        (function(fn, thisArg, args) {
            var box = { done: false, value: undefined, error: undefined };
            fn.apply(thisArg, args).then(
                function(v) { box.done = true; box.value = v; },
                function(e) { box.done = true; box.error = (e && e.message) ? e.message : String(e); }
            );
            return box;
        })
        """.trimIndent(),
    )

    private fun callMethod(methodName: String, vararg args: Any?): Value {
        val method = pluginValue.getMember(methodName)
            ?: throw NovelPluginException("$id: plugin has no method $methodName")
        val box = callAsync.execute(method, pluginValue, args.toList().toTypedArray())
        if (!box.getMember("done").asBoolean()) {
            throw NovelPluginException(
                "$id: $methodName did not resolve synchronously " +
                    "(plugin likely used a real timer/delay, which this bridge doesn't support yet)",
            )
        }
        val error = box.getMember("error")
        if (error != null && !error.isNull) {
            throw NovelPluginException("$id: $methodName threw: ${error.asString()}")
        }
        return box.getMember("value")
    }

    fun popularNovels(pageNo: Int): List<NovelItem> =
        toValueList(callMethod("popularNovels", pageNo, emptyOptionsValue())).map(::toNovelItem)

    private fun emptyOptionsValue(): Value =
        context.eval("js", "({ filters: {}, showLatestNovels: false })")

    fun searchNovels(searchTerm: String, pageNo: Int): List<NovelItem> =
        toValueList(callMethod("searchNovels", searchTerm, pageNo)).map(::toNovelItem)

    fun parseNovel(novelPath: String): SourceNovel =
        toSourceNovel(callMethod("parseNovel", novelPath))

    fun parseChapter(chapterPath: String): String =
        callMethod("parseChapter", chapterPath).asString()

    override fun close() = context.close(true)

    private fun toValueList(v: Value): List<Value> = (0 until v.arraySize).map { v.getArrayElement(it) }

    private fun toNovelItem(v: Value) = NovelItem(
        name = v.str("name") ?: "",
        path = v.str("path") ?: "",
        cover = v.str("cover"),
    )

    private fun toChapterItem(v: Value) = ChapterItem(
        name = v.str("name") ?: "",
        path = v.str("path") ?: "",
        chapterNumber = v.dbl("chapterNumber"),
        releaseTime = v.str("releaseTime"),
        page = v.str("page"),
    )

    private fun toSourceNovel(v: Value) = SourceNovel(
        name = v.str("name") ?: "",
        path = v.str("path") ?: "",
        cover = v.str("cover"),
        genres = v.str("genres"),
        summary = v.str("summary"),
        author = v.str("author"),
        artist = v.str("artist"),
        status = v.str("status"),
        chapters = v.getMember("chapters")?.let { toValueList(it).map(::toChapterItem) } ?: emptyList(),
        totalPages = v.dbl("totalPages")?.toInt(),
    )
}
