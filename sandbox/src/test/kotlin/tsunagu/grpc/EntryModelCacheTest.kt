package tsunagu.grpc

import eu.kanade.tachiyomi.source.model.SChapter
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertSame

class EntryModelCacheTest {
    private fun chapter(series: String) = SChapter.create().apply {
        url = "chapter-92"
        name = "Chapter 92"
        memo = JsonObject(mapOf("mangaPath" to JsonPrimitive(series)))
    }

    // LHTranslation-style: the same bare chapter url in two series, told apart only by memo.
    @Test
    fun sameChapterUrlInTwoSeriesStaysApart() {
        MemoStore.baseDir = Files.createTempDirectory("memo").toFile().also { it.deleteOnExit() }
        val a = chapter("/manga/a/")
        val b = chapter("/manga/b/")
        EntryModelCache.putChapter("lh", "/manga/a/", a)
        EntryModelCache.putChapter("lh", "/manga/b/", b)

        assertSame(a, EntryModelCache.getChapter("lh", "/manga/a/", "chapter-92"))
        assertSame(b, EntryModelCache.getChapter("lh", "/manga/b/", "chapter-92"))
        assertEquals(a.memo, EntryModelCache.chapterMemo("lh", "/manga/a/", "chapter-92"))
        assertEquals(b.memo, EntryModelCache.chapterMemo("lh", "/manga/b/", "chapter-92"))
    }
}
