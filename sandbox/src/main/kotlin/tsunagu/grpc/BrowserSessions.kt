package tsunagu.grpc

import eu.kanade.tachiyomi.network.BrowserCookie
import eu.kanade.tachiyomi.network.BrowserState
import eu.kanade.tachiyomi.network.PersistentCookieStore
import kotlinx.serialization.json.Json
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

/**
 * Moves a browser session between a real browser (where the user passed a
 * challenge) and the cookie store sources use. Only cookies that belong to
 * the URL's host go in or come out.
 */
object BrowserSessions {
    private const val MAX_STATE_BYTES = 256 * 1024
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    /** Applies [stateJson] when [apply] is set, then returns the host's session as JSON. */
    fun exchange(
        store: PersistentCookieStore,
        url: String,
        stateJson: String,
        apply: Boolean,
    ): String {
        val target = url.toHttpUrlOrNull()
        require(target != null && target.scheme in setOf("http", "https")) { "invalid URL" }
        if (apply) {
            require(stateJson.length <= MAX_STATE_BYTES) { "browser state too large" }
            val state = json.decodeFromString<BrowserState>(stateJson)
            val host = target.host
            val cookies =
                state.cookies.map { c ->
                    val belongs =
                        if (c.hostOnly) c.domain == host else host == c.domain || host.endsWith("." + c.domain)
                    require(belongs) { "cookie for ${c.domain} does not belong to $host" }
                    c.toCookie()
                }
            store.addAll(target, cookies)
            store.setUserAgent(target, state.userAgent)
        }
        val current = BrowserState(store.get(target).map(BrowserCookie::from), store.userAgent(target).orEmpty())
        return json.encodeToString(BrowserState.serializer(), current)
    }
}
