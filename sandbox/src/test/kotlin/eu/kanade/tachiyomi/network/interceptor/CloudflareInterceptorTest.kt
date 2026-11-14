package eu.kanade.tachiyomi.network.interceptor

import com.sun.net.httpserver.HttpServer
import eu.kanade.tachiyomi.network.PersistentCookieJar
import eu.kanade.tachiyomi.network.PersistentCookieStore
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Request
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicInteger
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals

class CloudflareInterceptorTest {
    private val solves = AtomicInteger()
    private var solverPage = "<html><head></head><body>solved page</body></html>"
    private val solver = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0).apply {
        createContext("/v1") { ex ->
            solves.incrementAndGet()
            val body = buildJsonObject {
                put("status", "ok")
                putJsonObject("solution") {
                    put("url", "https://site.test/comics/x-123")
                    put("status", 200)
                    put("response", solverPage)
                    put("userAgent", "SolverAgent")
                    put("cookies", buildJsonArray { add(buildJsonObject { put("name", "cf_clearance"); put("value", "ok"); put("domain", ".site.test") }) })
                }
            }.let { Json.encodeToString(it) }.toByteArray()
            ex.sendResponseHeaders(200, body.size.toLong())
            ex.responseBody.use { it.write(body) }
        }
        start()
    }

    @AfterTest
    fun stop() = solver.stop(0)

    // Stands in for the site: challenges until [passAfter] attempts, recording the agent and the cookies the jar would send.
    private val store = PersistentCookieStore()

    private inner class Site(val passAfter: Int = Int.MAX_VALUE) : Interceptor {
        val hits = AtomicInteger()
        val seen = mutableListOf<Pair<String?, String?>>()

        override fun intercept(chain: Interceptor.Chain): Response {
            val req = chain.request()
            seen += req.header("User-Agent") to store.get(req.url).joinToString("; ") { "${it.name}=${it.value}" }
            val pass = hits.incrementAndGet() > passAfter
            return Response.Builder().request(req).protocol(Protocol.HTTP_1_1)
                .code(if (pass) 200 else 403).message("")
                .header("Server", "cloudflare")
                .apply { if (!pass) header("cf-mitigated", "challenge") }
                .body((if (pass) "origin page" else "challenge").toResponseBody("text/html".toMediaType()))
                .build()
        }
    }

    private fun client(site: Site): OkHttpClient {
        return OkHttpClient.Builder()
            .cookieJar(PersistentCookieJar(store))
            .addInterceptor(CloudflareInterceptor(store, "http://127.0.0.1:${solver.address.port}"))
            .addInterceptor(site)
            .build()
    }

    private fun OkHttpClient.get(url: String, accept: String? = null) =
        newCall(Request.Builder().url(url).apply { accept?.let { header("Accept", it) } }.build()).execute()

    @Test
    fun replaysTheClearanceWhenTheSiteAcceptsIt() {
        val site = Site(passAfter = 1)
        client(site).get("https://site.test/comics/x").use {
            assertEquals(200, it.code)
            assertEquals("origin page", it.body.string())
        }
        assertEquals(1, solves.get())
        assertEquals("SolverAgent" to "cf_clearance=ok", site.seen[1])
    }

    @Test
    fun servesTheSolversPageWhenTheClearanceIsRejected() {
        val site = Site()
        val http = client(site)
        http.get("https://site.test/comics/x").use {
            assertEquals(200, it.code)
            assertEquals("<html><head></head><body>solved page</body></html>", it.body.string())
            assertEquals("https://site.test/comics/x-123", it.request.url.toString())
        }
        assertEquals(1, solves.get())
        assertEquals(2, site.hits.get())

        // The host is known to reject replays now, so the next page skips the doomed retry.
        http.get("https://site.test/comics/y").use { assertEquals(200, it.code) }
        assertEquals(2, solves.get())
        assertEquals(3, site.hits.get())
    }

    @Test
    fun unwrapsJsonThatChromeShowsAsText() {
        solverPage = """<html><head><meta name="color-scheme" content="light dark"><meta charset="utf-8"></head><body><pre>{"a":"&lt;b&gt; &amp; c"}</pre><div class="json-formatter-container"></div></body></html>"""
        client(Site()).get("https://site.test/api/x").use {
            assertEquals("""{"a":"<b> & c"}""", it.body.string())
            assertEquals("application/json; charset=utf-8", it.header("Content-Type"))
        }
    }

    @Test
    fun leavesImagesChallenged() {
        client(Site()).get("https://site.test/cover.webp").use { assertEquals(403, it.code) }
    }
}
