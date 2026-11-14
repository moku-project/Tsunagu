package eu.kanade.tachiyomi.network

import okhttp3.Cookie
import okhttp3.HttpUrl.Companion.toHttpUrl
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class PersistentCookieStoreTest {
    private val source = "https://source.example/".toHttpUrl()

    private fun cookie(
        name: String,
        domain: String,
        hostOnly: Boolean = true,
        path: String = "/",
        expiresAt: Long = System.currentTimeMillis() + 60_000,
    ) = Cookie.Builder().name(name).value("v").path(path).expiresAt(expiresAt)
        .apply { if (hostOnly) hostOnlyDomain(domain) else domain(domain) }.build()

    @Test
    fun doesNotLeakToHostsThatMerelyEndTheSame() {
        val store = PersistentCookieStore()
        store.addAll(source, listOf(cookie("cf_clearance", "source.example")))
        assertTrue(store.get("https://evilsource.example/".toHttpUrl()).isEmpty())
        assertEquals(1, store.get(source).size)
    }

    @Test
    fun honoursHostOnlyPathAndExpiry() {
        val store = PersistentCookieStore()
        store.addAll(
            source,
            listOf(
                cookie("host", "source.example"),
                cookie("wide", "source.example", hostOnly = false),
                cookie("scoped", "source.example", path = "/reader"),
                cookie("gone", "source.example", expiresAt = System.currentTimeMillis() - 1),
            ),
        )
        assertEquals(setOf("host", "wide"), store.get(source).map { it.name }.toSet())
        assertEquals(setOf("wide"), store.get("https://cdn.source.example/".toHttpUrl()).map { it.name }.toSet())
        assertEquals(
            setOf("host", "wide", "scoped"),
            store.get("https://source.example/reader/1".toHttpUrl()).map { it.name }.toSet(),
        )
    }

    @Test
    fun persistsCookiesAndAgentsButNotSessionCookies() {
        val file = Files.createTempDirectory("cookies").resolve("cookies.json").toFile()
        val first = PersistentCookieStore(file)
        first.addAll(source, listOf(cookie("kept", "source.example")))
        first.addAll(source, listOf(Cookie.Builder().name("session").value("v").hostOnlyDomain("source.example").build()))
        first.setUserAgent(source, "Solver Browser")

        val second = PersistentCookieStore(file)
        assertEquals(listOf("kept"), second.get(source).map { it.name })
        assertEquals("Solver Browser", second.userAgent(source))
        assertNull(second.userAgent("https://other.example/".toHttpUrl()))
    }
}
