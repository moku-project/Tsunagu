package tsunagu.net

import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class NativeHttpTest {
    private val sent = mutableListOf<Pair<String, Map<String, String>>>()
    private val jar = object : CookieJar {
        val cookies = mutableListOf<Cookie>()
        override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) { this.cookies += cookies }
        override fun loadForRequest(url: HttpUrl) = cookies.filter { it.matches(url) }
    }

    @BeforeTest
    fun setUp() {
        System.setProperty("tsunagu.nativeHttp", "true")
        NativeHttp.send = { method, url, headers, _, _, _ ->
            sent += url to headers.toList().chunked(2).associate { it[0] to it[1] }
            when {
                url.endsWith("/start") -> arrayOf(302, arrayOf("Location", "/end", "Set-Cookie", "session=abc; Path=/"), ByteArray(0))
                else -> arrayOf(200, arrayOf("Content-Type", "text/plain"), "hello $method".toByteArray())
            }
        }
    }

    @AfterTest
    fun tearDown() {
        System.clearProperty("tsunagu.nativeHttp")
    }

    @Test
    fun followsRedirectsWithCookiesAndRunsLaterInterceptors() {
        val base = OkHttpClient.Builder().cookieJar(jar).addInterceptor(NativeHttp.interceptor).build()
        var extensionRan = false
        val client = base.newBuilder().addInterceptor { extensionRan = true; it.proceed(it.request().newBuilder().header("X-Ext", "1").build()) }.build()
        val response = client.newCall(Request.Builder().url("https://example.com/start").build()).execute()
        assertEquals(200, response.code)
        assertEquals("hello GET", response.body.string())
        assertEquals("https://example.com/end", response.request.url.toString())
        assertTrue(extensionRan)
        assertEquals(listOf("https://example.com/start", "https://example.com/end"), sent.map { it.first })
        assertEquals("session=abc", sent[1].second["Cookie"])
        assertEquals("1", sent[0].second["X-Ext"])
        assertEquals(302, response.priorResponse?.code)
    }

    @Test
    fun leavesPlainHttpToOkHttpWhenDisabled() {
        System.clearProperty("tsunagu.nativeHttp")
        val client = OkHttpClient.Builder().addInterceptor(NativeHttp.interceptor)
            .addInterceptor { okhttp3.Response.Builder().request(it.request()).protocol(okhttp3.Protocol.HTTP_1_1).code(204).message("").build() }
            .build()
        assertEquals(204, client.newCall(Request.Builder().url("https://example.com/").build()).execute().code)
        assertTrue(sent.isEmpty())
    }
}
