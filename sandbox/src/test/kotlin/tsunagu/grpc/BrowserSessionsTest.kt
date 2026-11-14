package tsunagu.grpc

import eu.kanade.tachiyomi.network.PersistentCookieStore
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class BrowserSessionsTest {
    private fun state(domain: String) =
        """{"userAgent":"Acceptance Browser","cookies":[{"name":"cf_clearance","value":"ok","domain":"$domain",""" +
            """"hostOnly":true,"secure":true,"expiresAt":${System.currentTimeMillis() + 600_000}}]}"""

    @Test
    fun appliesAndReturnsTheHostsSessionOnly() {
        val store = PersistentCookieStore()
        val saved = Json.parseToJsonElement(BrowserSessions.exchange(store, "https://source.example/", state("source.example"), true)).jsonObject
        assertEquals("Acceptance Browser", saved["userAgent"]!!.jsonPrimitive.content)
        val cookie = saved["cookies"]!!.jsonArray.single().jsonObject
        assertEquals("/", cookie["path"]!!.jsonPrimitive.content)

        val other = Json.parseToJsonElement(BrowserSessions.exchange(store, "https://evilsource.example/", "", false)).jsonObject
        assertTrue(other["cookies"]!!.jsonArray.isEmpty())
        assertEquals("", other["userAgent"]!!.jsonPrimitive.content)
    }

    @Test
    fun rejectsCookiesForAnotherDomain() {
        assertFailsWith<IllegalArgumentException> {
            BrowserSessions.exchange(PersistentCookieStore(), "https://source.example/", state("tracker.example"), true)
        }
    }

    @Test
    fun rejectsNonHttpUrls() {
        assertFailsWith<IllegalArgumentException> {
            BrowserSessions.exchange(PersistentCookieStore(), "file:///etc/passwd", "", false)
        }
    }
}
