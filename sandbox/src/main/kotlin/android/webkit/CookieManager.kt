package android.webkit

import java.util.concurrent.ConcurrentHashMap

class CookieManager private constructor() {
    private val cookies = ConcurrentHashMap<String, String>()

    fun setAcceptCookie(accept: Boolean) {  }

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

    fun removeSessionCookie() {  }

    fun hasCookies(): Boolean = cookies.isNotEmpty()

    fun flush() {  }

    companion object {
        private val instance = CookieManager()

        @JvmStatic
        fun getInstance(): CookieManager = instance
    }
}
