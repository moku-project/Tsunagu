// Adapted from Mihon/Tachiyomi (https://github.com/mihonapp/mihon), licensed under Apache License 2.0.
// See /NOTICE.md and /THIRD_PARTY_LICENSES/Apache-2.0.txt for full attribution.

package eu.kanade.tachiyomi.network

import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.Cookie
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okio.withLock
import java.io.File
import java.net.CookieStore
import java.net.HttpCookie
import java.net.URI
import java.util.concurrent.locks.ReentrantLock
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds

/**
 * Cookies and per-host User-Agents that sources send.
 *
 * Cookies are matched with OkHttp's own rules (domain, host-only, path,
 * expiry), so one host never receives another's cookies. A challenge
 * clearance (e.g. Cloudflare's cf_clearance) is bound to the User-Agent that
 * earned it, so that agent is stored for the host too.
 *
 * With [file], persistent cookies and agents survive restarts.
 */
class PersistentCookieStore(private val file: File? = null) : CookieStore {
    private val cookies = mutableListOf<Cookie>()
    private val agents = mutableMapOf<String, String>()
    private val lock = ReentrantLock()

    init {
        load()
    }

    fun addAll(
        url: HttpUrl,
        cookies: List<Cookie>,
    ) {
        if (cookies.isEmpty()) return
        lock.withLock {
            cookies.forEach(::putLocked)
            persistLocked()
        }
    }

    fun get(url: HttpUrl): List<Cookie> =
        lock.withLock {
            val now = System.currentTimeMillis()
            if (cookies.removeAll { it.expiresAt <= now }) persistLocked()
            cookies.filter { it.matches(url) }
        }

    /** The User-Agent a challenge on this host was solved with, if any. */
    fun userAgent(url: HttpUrl): String? = lock.withLock { agents[url.host] }

    fun setUserAgent(
        url: HttpUrl,
        agent: String,
    ) {
        require(agent.length <= 2048 && agent.none { it == '\r' || it == '\n' }) { "invalid User-Agent" }
        lock.withLock {
            if (agent.isEmpty()) agents.remove(url.host) else agents[url.host] = agent
            persistLocked()
        }
    }

    override fun removeAll(): Boolean =
        lock.withLock {
            val hadAny = cookies.isNotEmpty() || agents.isNotEmpty()
            cookies.clear()
            agents.clear()
            persistLocked()
            hadAny
        }

    fun remove(uri: URI) {
        val host = uri.host ?: return
        lock.withLock {
            cookies.removeAll { it.domain == host }
            agents.remove(host)
            persistLocked()
        }
    }

    override fun get(uri: URI): List<HttpCookie> {
        val url = uri.toString().toHttpUrlOrNull() ?: return emptyList()
        return get(url).map { it.toHttpCookie() }
    }

    override fun add(
        uri: URI?,
        cookie: HttpCookie,
    ) {
        val parsed = cookie.toCookie(uri?.host) ?: return
        lock.withLock {
            putLocked(parsed)
            persistLocked()
        }
    }

    override fun getCookies(): List<HttpCookie> = getStoredCookies().map { it.toHttpCookie() }

    fun getStoredCookies(): List<Cookie> = lock.withLock { cookies.toList() }

    override fun getURIs(): List<URI> = getStoredCookies().map { URI("http://${it.domain}") }.distinct()

    override fun remove(
        uri: URI?,
        cookie: HttpCookie,
    ): Boolean {
        val target = cookie.toCookie(uri?.host) ?: return false
        return lock.withLock {
            val removed = cookies.removeAll { it.sameSlot(target) }
            if (removed) persistLocked()
            removed
        }
    }

    // One cookie per (name, domain, path), as browsers keep them.
    private fun putLocked(cookie: Cookie) {
        cookies.removeAll { it.sameSlot(cookie) }
        if (cookie.expiresAt > System.currentTimeMillis()) cookies.add(cookie)
    }

    private fun Cookie.sameSlot(other: Cookie) = name == other.name && domain == other.domain && path == other.path

    private fun load() {
        val source = file?.takeIf { it.isFile } ?: return
        runCatching {
            val saved = json.decodeFromString<Saved>(source.readText())
            val now = System.currentTimeMillis()
            cookies += saved.cookies.map { it.toCookie() }.filter { it.expiresAt > now }
            agents += saved.agents
        }.onFailure { logger.warn(it) { "ignoring unreadable cookie file ${source.path}" } }
    }

    private fun persistLocked() {
        val target = file ?: return
        runCatching {
            target.parentFile?.mkdirs()
            val temp = File(target.path + ".tmp")
            temp.writeText(json.encodeToString(Saved(cookies.filter { it.persistent }.map(BrowserCookie::from), agents.toMap())))
            if (!temp.renameTo(target)) error("could not replace ${target.path}")
        }.onFailure { logger.warn(it) { "could not save cookies" } }
    }

    @Serializable
    private data class Saved(
        val cookies: List<BrowserCookie> = emptyList(),
        val agents: Map<String, String> = emptyMap(),
    )

    private fun HttpCookie.toCookie(urlDomain: String?): Cookie? {
        return Cookie
            .Builder()
            .name(name)
            .value(value)
            .domain((domain ?: urlDomain ?: return null).removePrefix("."))
            .path(path ?: "/")
            .also {
                if (maxAge != -1L) {
                    it.expiresAt(System.currentTimeMillis() + maxAge.seconds.inWholeMilliseconds)
                } else {
                    it.expiresAt(Long.MAX_VALUE)
                }
                if (secure) {
                    it.secure()
                }
                if (isHttpOnly) {
                    it.httpOnly()
                }
                if (domain != null && !domain.startsWith('.')) {
                    it.hostOnlyDomain(domain.removePrefix("."))
                }
            }.build()
    }

    private fun Cookie.toHttpCookie(): HttpCookie {
        val it = this
        return HttpCookie(it.name, it.value).apply {
            domain =
                if (hostOnly) {
                    it.domain
                } else {
                    "." + it.domain
                }
            path = it.path
            secure = it.secure
            maxAge =
                if (it.persistent) {
                    (it.expiresAt.milliseconds - System.currentTimeMillis().milliseconds).inWholeSeconds
                } else {
                    -1
                }

            isHttpOnly = it.httpOnly
        }
    }

    private companion object {
        val logger = KotlinLogging.logger {}
        val json = Json { ignoreUnknownKeys = true }
    }
}

/** A cookie as browsers and the BrowserCookies RPC describe it. */
@Serializable
data class BrowserCookie(
    val name: String,
    val value: String,
    val domain: String,
    val path: String = "/",
    val expiresAt: Long = Long.MAX_VALUE,
    val secure: Boolean = false,
    val httpOnly: Boolean = false,
    val hostOnly: Boolean = true,
    val persistent: Boolean = true,
) {
    fun toCookie(): Cookie =
        Cookie
            .Builder()
            .name(name)
            .value(value)
            .path(path)
            .apply {
                if (hostOnly) hostOnlyDomain(domain) else domain(domain)
                if (persistent) expiresAt(expiresAt)
                if (secure) secure()
                if (httpOnly) httpOnly()
            }.build()

    companion object {
        fun from(c: Cookie) = BrowserCookie(c.name, c.value, c.domain, c.path, c.expiresAt, c.secure, c.httpOnly, c.hostOnly, c.persistent)
    }
}

/** A host's browser session: what a real browser presents after passing a challenge. */
@Serializable
data class BrowserState(
    val cookies: List<BrowserCookie> = emptyList(),
    val userAgent: String = "",
)
