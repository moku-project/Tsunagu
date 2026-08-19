package eu.kanade.tachiyomi.network

import okhttp3.Cookie
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okio.withLock
import java.net.CookieStore
import java.net.HttpCookie
import java.net.URI
import java.util.concurrent.locks.ReentrantLock
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds

class PersistentCookieStore : CookieStore {
    private val cookieMap = mutableMapOf<String, List<Cookie>>()

    private val lock = ReentrantLock()

    fun addAll(
        url: HttpUrl,
        cookies: List<Cookie>,
    ) {
        lock.withLock {
            for (cookie in cookies) {
                val cookiesForDomain = cookieMap[cookie.domain].orEmpty().toMutableList()
                val pos = cookiesForDomain.indexOfFirst { it.name == cookie.name }
                if (pos == -1) {
                    cookiesForDomain.add(cookie)
                } else {
                    cookiesForDomain[pos] = cookie
                }
                cookieMap[cookie.domain] = cookiesForDomain
            }
        }
    }

    override fun removeAll(): Boolean =
        lock.withLock {
            val wasNotEmpty = cookieMap.isNotEmpty()
            cookieMap.clear()
            wasNotEmpty
        }

    fun remove(uri: URI) {
        val url = uri.toURL()
        lock.withLock {
            cookieMap.remove(url.host)
        }
    }

    override fun get(uri: URI): List<HttpCookie> {
        val url = uri.toURL()
        return get(url.toHttpUrlOrNull()!!).map {
            it.toHttpCookie()
        }
    }

    fun get(url: HttpUrl): List<Cookie> =
        lock.withLock {
            cookieMap.entries
                .filter {
                    url.host.endsWith(it.key)
                }.flatMap { it.value }
        }

    override fun add(
        uri: URI?,
        cookie: HttpCookie,
    ) {
        lock.withLock {
            val cookie = cookie.toCookie(uri?.host) ?: return@withLock
            val cookiesForDomain = cookieMap[cookie.domain].orEmpty().toMutableList()
            val pos = cookiesForDomain.indexOfFirst { it.name == cookie.name }
            if (pos == -1) {
                cookiesForDomain.add(cookie)
            } else {
                cookiesForDomain[pos] = cookie
            }
            cookieMap[cookie.domain] = cookiesForDomain
        }
    }

    override fun getCookies(): List<HttpCookie> =
        lock.withLock {
            cookieMap.values.flatMap {
                it.map { c -> c.toHttpCookie() }
            }
        }

    fun getStoredCookies(): List<Cookie> =
        lock.withLock {
            cookieMap.values.flatMap { it }
        }

    override fun getURIs(): List<URI> =
        lock.withLock {
            cookieMap.keys.toList().map {
                URI("http://$it")
            }
        }

    override fun remove(
        uri: URI?,
        cookie: HttpCookie,
    ): Boolean =
        lock.withLock {
            val cookie = cookie.toCookie(uri?.host) ?: return@withLock false
            val cookies = cookieMap[cookie.domain].orEmpty()
            val index =
                cookies.indexOfFirst {
                    it.name == cookie.name &&
                        it.path == cookie.path
                }
            if (index >= 0) {
                val newList = cookies.toMutableList()
                newList.removeAt(index)
                cookieMap[cookie.domain] = newList.toList()
                true
            } else {
                false
            }
        }

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
                    -1
                } else {
                    (it.expiresAt.milliseconds - System.currentTimeMillis().milliseconds).inWholeSeconds
                }

            isHttpOnly = it.httpOnly
        }
    }
}
