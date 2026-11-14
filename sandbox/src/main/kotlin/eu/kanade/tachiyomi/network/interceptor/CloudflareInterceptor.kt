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
import okhttp3.Protocol
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import java.io.IOException
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

class CloudflareInterceptor(
    private val cookieStore: PersistentCookieStore,
    private val solverUrl: String = System.getenv("SANDBOX_FLARESOLVERR_URL")?.trim()?.trimEnd('/').orEmpty(),
) : Interceptor {
    private val logger = KotlinLogging.logger {}
    private val json = Json { ignoreUnknownKeys = true }

    private val solvedUserAgents = ConcurrentHashMap<String, SolvedAgent>()
    private val hostLocks = ConcurrentHashMap<String, Any>()

    // Hosts that still challenge this client after a solve; their pages come straight from the solver.
    private val solverOnlyHosts = ConcurrentHashMap.newKeySet<String>()

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

        // Not a challenge: the origin itself refused. Some embed hosts (e.g. anidrive.click)
        // only serve requests that look like a browser navigation, so retry once as one.
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
        val servable = request.method == "GET" && !request.wantsImage()
        if (servable && host in solverOnlyHosts) return solverPage(request, solveAndStore(request))

        var fresh: Solved? = null
        // Concurrent requests to a challenged host wait for one solve instead of each starting their own.
        val agent = synchronized(hostLocks.computeIfAbsent(host) { Any() }) {
            val known = solvedUserAgents[host]
            if (known != null && known.at >= started) {
                known
            } else {
                logger.info { "Cloudflare challenge on $host; solving via FlareSolverr" }
                val solved = solveAndStore(request).also { fresh = it }
                SolvedAgent(solved.userAgent, System.nanoTime()).also { solvedUserAgents[host] = it }
            }
        }
        // cf_clearance is bound to the solver's User-Agent, so the retry must send it.
        val retried = chain.proceed(request.newBuilder().header("User-Agent", agent.userAgent).build())
        if (!servable || !retried.isCloudflareChallenge()) return retried

        // Cloudflare can reject this client's TLS fingerprint even with a valid clearance, so use the solver's own copy.
        retried.close()
        logger.info { "Cloudflare still challenges $host after solving; serving its pages from FlareSolverr" }
        solverOnlyHosts.add(host)
        return solverPage(request, fresh ?: solveAndStore(request))
    }

    private fun solveAndStore(request: Request): Solved {
        val solved = solve(request.url.toString())
            ?: throw IOException("FlareSolverr could not solve the Cloudflare challenge on ${request.url.host}")
        cookieStore.addAll(request.url, solved.cookies)
        return solved
    }

    private fun solverPage(request: Request, solved: Solved): Response {
        if (solved.html.isEmpty()) throw IOException("FlareSolverr returned no page for ${request.url.host}")
        // Chrome shows JSON and plain text inside a <pre>, so unwrap it back to the raw body.
        val raw = CHROME_TEXT_VIEW.find(solved.html)?.groupValues?.get(1)?.let(::unescapeHtml)
        val type = when {
            raw == null -> HTML_MEDIA
            raw.trimStart().startsWith('{') || raw.trimStart().startsWith('[') -> JSON_MEDIA
            else -> TEXT_MEDIA
        }
        return Response.Builder()
            .request(request.newBuilder().url(solved.url.toHttpUrlOrNull() ?: request.url).build())
            .protocol(Protocol.HTTP_1_1)
            .code(solved.status)
            .message("")
            .header("Content-Type", type.toString())
            .body((raw ?: solved.html).toResponseBody(type))
            .build()
    }

    private fun Response.isCloudflareChallenge() =
        code in ERROR_CODES && header("Server") in SERVER_CHECK && isChallenge(this)

    private fun Request.wantsImage() =
        header("Accept")?.startsWith("image/") == true ||
            url.encodedPath.substringAfterLast('.', "").lowercase() in IMAGE_EXTENSIONS

    private fun unescapeHtml(s: String) =
        s.replace("&lt;", "<").replace("&gt;", ">").replace("&quot;", "\"").replace("&#39;", "'")
            .replace("&nbsp;", " ").replace("&amp;", "&")

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

    private data class Solved(
        val cookies: List<Cookie>,
        val userAgent: String,
        val url: String,
        val status: Int,
        val html: String,
    )

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
            val s = parsed.solution
            return Solved(cookies, s.userAgent, s.url.ifEmpty { targetUrl }, s.status.takeIf { it in 100..599 } ?: 200, s.response)
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
        val status: Int = 0,
        val response: String = "",
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
        private val HTML_MEDIA = "text/html; charset=utf-8".toMediaType()
        private val TEXT_MEDIA = "text/plain; charset=utf-8".toMediaType()
        private val CHROME_TEXT_VIEW = Regex("""^<html><head>.{0,1000}?</head><body><pre[^>]*>(.*)</pre>""", RegexOption.DOT_MATCHES_ALL)
        private val IMAGE_EXTENSIONS = setOf("jpg", "jpeg", "png", "gif", "webp", "avif", "bmp", "svg")
    }
}
