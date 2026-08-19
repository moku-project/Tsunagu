package android.webkit

import java.util.concurrent.ConcurrentHashMap

class CookieManager private constructor() {
    private val cookies = ConcurrentHashMap<String, String>()

    fun setAcceptCookie(accept: Boolean) { /* no-op, no WebView to configure */ }

    fun acceptCookie(): Boolean = true

    fun setCookie(url: String?, value: String?) {
        if (url == null || value == null) return
        cookies[url] = value
    }

    fun getCookie(url: String?): String? = url?.let { cookies[it] }

    fun removeAllCookie() {
        cookies.clear()
    }

    fun removeAllCookies(callback: ValueCallback<Boolean>?) {
        cookies.clear()
        callback?.onReceiveValue(true)
    }

    fun removeSessionCookie() { /* no persistence distinction in this stub */ }

    fun hasCookies(): Boolean = cookies.isNotEmpty()

    fun flush() { /* no-op, nothing to flush */ }

    companion object {
        private val instance = CookieManager()

        @JvmStatic
        fun getInstance(): CookieManager = instance
    }
}
