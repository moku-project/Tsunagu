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
import java.util.concurrent.TimeUnit

class CloudflareInterceptor(
    private val cookieStore: PersistentCookieStore,
) : Interceptor {
    private val logger = KotlinLogging.logger {}
    private val json = Json { ignoreUnknownKeys = true }

    private val solverUrl = System.getenv("SANDBOX_FLARESOLVERR_URL")?.trim()?.trimEnd('/').orEmpty()

    private val solverClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(90, TimeUnit.SECONDS)
            .callTimeout(100, TimeUnit.SECONDS)
            .build()
    }

    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val response = chain.proceed(request)

        if (response.code !in ERROR_CODES || response.header("Server") !in SERVER_CHECK) {
            return response
        }

        val host = request.url.host
        if (solverUrl.isEmpty()) {
            response.close()
            throw IOException("Cloudflare challenge on $host; set flaresolverr_url in settings to bypass it")
        }

        response.close()
        logger.info { "Cloudflare challenge on $host; solving via FlareSolverr" }

        val solved = solve(request.url.toString())
            ?: throw IOException("FlareSolverr could not solve the Cloudflare challenge on $host")

        cookieStore.addAll(request.url, solved.cookies)
        val retried = request.newBuilder().header("User-Agent", solved.userAgent).build()
        return chain.proceed(retried)
    }

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
        private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()
    }
}
