package tsunagu.source

import eu.kanade.tachiyomi.source.model.Page
import eu.kanade.tachiyomi.source.online.HttpSource
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import tsunagu.loader.ContentType
import tsunagu.loader.ContentTypeClassifier
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

private open class MangaStub : HttpSource() {
    override val name = "stub"
    override val lang = "en"
    override val baseUrl = "https://example.com"
    override val supportsLatest = false
}

// Like a ProGuarded Tsundoku extension: the method is there, the NovelSource interface isn't.
private class NovelStub : MangaStub() {
    @Suppress("unused")
    suspend fun fetchPageText(page: Page): String {
        delay(1) // really suspend, so the reflective call must handle COROUTINE_SUSPENDED
        if (page.url == "boom") throw IllegalStateException("captcha")
        return "<p>${page.index}</p>"
    }
}

class NovelSourcesTest {
    @Test
    fun classifiesBySignature() {
        assertEquals(ContentType.NOVEL, ContentTypeClassifier.classify(NovelStub::class.java))
        assertEquals(ContentType.MANGA, ContentTypeClassifier.classify(MangaStub::class.java))
    }

    @Test
    fun joinsPagesInOrder() {
        val text = runBlocking { NovelSources.chapterText(NovelStub(), listOf(Page(0, "a"), Page(1, "b"))) }
        assertEquals("<p>0</p>\n<p>1</p>", text)
    }

    @Test
    fun propagatesSourceErrors() {
        val e = assertFailsWith<IllegalStateException> {
            runBlocking { NovelSources.chapterText(NovelStub(), listOf(Page(0, "boom"))) }
        }
        assertEquals("captcha", e.message)
    }
}
