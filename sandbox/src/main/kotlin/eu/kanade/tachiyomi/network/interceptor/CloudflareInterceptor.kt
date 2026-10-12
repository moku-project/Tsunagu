// Adapted from Mihon/Tachiyomi (https://github.com/mihonapp/mihon), licensed under Apache License 2.0.
// See /NOTICE.md and /THIRD_PARTY_LICENSES/Apache-2.0.txt for full attribution.

package eu.kanade.tachiyomi.network.interceptor

import eu.kanade.tachiyomi.network.PersistentCookieStore
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.Cookie
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

class CloudflareInterceptor(
    private val cookieStore: PersistentCookieStore,
) : Interceptor {
    private val logger = KotlinLogging.logger {}
    private val json = Json { ignoreUnknownKeys = true }

    private val solverUrl = System.getenv("SANDBOX_FLARESOLVERR_URL")?.trim()?.trimEnd('/').orEmpty()
    private val solvedUserAgents = ConcurrentHashMap<String, SolvedAgent>()
    private val hostLocks = ConcurrentHashMap<String, Any>()

    private val solverClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(90, TimeUnit.SECONDS)
            .callTimeout(100, TimeUnit.SECONDS)
            .build()
    }

    override fun intercept(chain: Interceptor.Chain): Response {
        val started = System.nanoTime()
        val original = chain.request()
        val host = original.url.host
        val request = solvedUserAgents[host]
            ?.let { original.newBuilder().header("User-Agent", it.userAgent).build() }
            ?: original
        val response = chain.proceed(request)

        if (response.code !in ERROR_CODES || response.header("Server") !in SERVER_CHECK) {
            return response
        }

        // Without cf-mitigated the block came from the origin, not a Cloudflare challenge.
        // Embed hosts (e.g. anidrive.click) reject requests that don't look like an iframe navigation.
        if (!isChallenge(response)) {
            if (response.code != 403 || request.header("Sec-Fetch-Mode") != null) return response
            response.close()
            return chain.proceed(withNavigationHeaders(request))
        }

        if (solverUrl.isEmpty()) {
            response.close()
            throw IOException("Cloudflare challenge on $host; set flaresolverr_url in settings to bypass it")
        }

        response.close()
        // Concurrent requests to a challenged host wait for one solve instead of each starting their own.
        val agent = synchronized(hostLocks.computeIfAbsent(host) { Any() }) {
            val known = solvedUserAgents[host]
            if (known != null && known.at >= started) {
                known
            } else {
                logger.info { "Cloudflare challenge on $host; solving via FlareSolverr" }
                val solved = solve(request.url.toString())
                    ?: throw IOException("FlareSolverr could not solve the Cloudflare challenge on $host")
                cookieStore.addAll(request.url, solved.cookies)
                SolvedAgent(solved.userAgent, System.nanoTime()).also { solvedUserAgents[host] = it }
            }
        }
        // cf_clearance is bound to the solver's User-Agent, so the retry must send it.
        return chain.proceed(request.newBuilder().header("User-Agent", agent.userAgent).build())
    }

    private fun isChallenge(response: Response): Boolean {
        if (response.header(CF_MITIGATED).equals("challenge", ignoreCase = true)) return true
        val head = runCatching { response.peekBody(CHALLENGE_PEEK_BYTES).string() }.getOrDefault("")
        return CHALLENGE_MARKERS.any { it in head }
    }

    private fun withNavigationHeaders(request: Request): Request {
        val referer = request.header("Referer")?.toHttpUrlOrNull()
        val site = when {
            referer == null -> "none"
            referer.host == request.url.host -> "same-origin"
            else -> "cross-site"
        }
        return request.newBuilder()
            .header("Accept", NAVIGATION_ACCEPT)
            .header("Sec-Fetch-Dest", if (site == "cross-site") "iframe" else "document")
            .header("Sec-Fetch-Mode", "navigate")
            .header("Sec-Fetch-Site", site)
            .header("Upgrade-Insecure-Requests", "1")
            .build()
    }

    private data class SolvedAgent(val userAgent: String, val at: Long)

    private data class Solved(val cookies: List<Cookie>, val userAgent: String)

    private fun solve(targetUrl: String): Solved? {
        val payload = json.encodeToString(
            V1Request.serializer(),
            V1Request(cmd = "request.get", url = targetUrl, maxTimeout = 60_000),
        ).toRequestBody(JSON_MEDIA)

        val call = Request.Builder().url("$solverUrl/v1").post(payload).build()
        solverClient.newCall(call).execute().use { resp ->
            val bodyText = resp.body?.string().orEmpty()
            if (!resp.isSuccessful) {
                logger.warn { "FlareSolverr HTTP ${resp.code}: ${bodyText.take(200)}" }
                return null
            }
            val parsed = runCatching { json.decodeFromString(V1Response.serializer(), bodyText) }.getOrNull()
            if (parsed?.status != "ok" || parsed.solution == null) {
                logger.warn { "FlareSolverr: ${parsed?.message ?: "unparseable response"}" }
                return null
            }
            val url = targetUrl.toHttpUrlOrNull() ?: return null
            val cookies = parsed.solution.cookies.mapNotNull { c ->
                if (c.name.isBlank()) return@mapNotNull null
                Cookie.Builder()
                    .name(c.name)
                    .value(c.value)
                    .domain(c.domain.removePrefix(".").ifEmpty { url.host })
                    .path(c.path.ifEmpty { "/" })
                    .let { if (c.secure) it.secure() else it }
                    .let { if (c.httpOnly) it.httpOnly() else it }
                    .build()
            }
            return Solved(cookies, parsed.solution.userAgent)
        }
    }

    @Serializable
    private data class V1Request(val cmd: String, val url: String, val maxTimeout: Int)

    @Serializable
    private data class V1Response(
        val status: String = "",
        val message: String = "",
        val solution: Solution? = null,
    )

    @Serializable
    private data class Solution(
        val url: String = "",
        val userAgent: String = "",
        val cookies: List<SolverCookie> = emptyList(),
    )

    @Serializable
    private data class SolverCookie(
        val name: String,
        val value: String,
        val domain: String = "",
        val path: String = "/",
        val secure: Boolean = false,
        val httpOnly: Boolean = false,
    )

    companion object {
        private val ERROR_CODES = listOf(403, 503)
        private val SERVER_CHECK = arrayOf("cloudflare-nginx", "cloudflare")
        private const val CF_MITIGATED = "cf-mitigated"
        private const val CHALLENGE_PEEK_BYTES = 64L * 1024
        private val CHALLENGE_MARKERS = listOf("_cf_chl_opt", "cf-browser-verification")
        private const val NAVIGATION_ACCEPT =
            "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
        private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()
    }
}
