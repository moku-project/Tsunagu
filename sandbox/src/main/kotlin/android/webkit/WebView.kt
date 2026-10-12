package android.webkit

import android.content.Context
import android.net.Uri
import android.view.View
import java.io.InputStream
import java.util.concurrent.Executors

open class WebSettings {
    var javaScriptEnabled: Boolean = false
    var domStorageEnabled: Boolean = false
    var userAgentString: String? = null
    var databaseEnabled: Boolean = false
    var blockNetworkImage: Boolean = false
    var loadsImagesAutomatically: Boolean = true
    var useWideViewPort: Boolean = false
    var loadWithOverviewMode: Boolean = false
    var cacheMode: Int = 0
    var mediaPlaybackRequiresUserGesture: Boolean = true
    var mixedContentMode: Int = 0

    fun setSupportZoom(support: Boolean) {}
    fun setSupportMultipleWindows(support: Boolean) {}
    fun setJavaScriptCanOpenWindowsAutomatically(flag: Boolean) {}
    fun setAppCacheEnabled(flag: Boolean) {}

    companion object {
        const val LOAD_DEFAULT = -1
        const val LOAD_NO_CACHE = 2
        const val LOAD_CACHE_ELSE_NETWORK = 1
        const val LOAD_CACHE_ONLY = 3
        const val MIXED_CONTENT_ALWAYS_ALLOW = 0

        @JvmStatic
        fun getDefaultUserAgent(context: Context?): String =
            "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 " +
                "(KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36"
    }
}

interface WebResourceRequest {
    fun getUrl(): Uri
    fun isForMainFrame(): Boolean
    fun isRedirect(): Boolean
    fun hasGesture(): Boolean
    fun getMethod(): String
    fun getRequestHeaders(): MutableMap<String, String>
}

interface WebResourceError {
    fun getErrorCode(): Int
    fun getDescription(): CharSequence
}

class WebResourceResponse {
    var mimeType: String?
    var encoding: String?
    var statusCode: Int = 200
    var reasonPhrase: String? = null
    var responseHeaders: MutableMap<String, String> = mutableMapOf()
    var data: InputStream?

    constructor(mimeType: String?, encoding: String?, data: InputStream?) {
        this.mimeType = mimeType
        this.encoding = encoding
        this.data = data
    }

    constructor(
        mimeType: String?,
        encoding: String?,
        statusCode: Int,
        reasonPhrase: String,
        responseHeaders: MutableMap<String, String>?,
        data: InputStream?,
    ) {
        this.mimeType = mimeType
        this.encoding = encoding
        this.statusCode = statusCode
        this.reasonPhrase = reasonPhrase
        this.responseHeaders = responseHeaders ?: mutableMapOf()
        this.data = data
    }
}

open class WebViewClient {
    open fun shouldInterceptRequest(view: WebView?, request: WebResourceRequest?): WebResourceResponse? = null
    open fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean = false
    open fun onPageStarted(view: WebView?, url: String?, favicon: Any?) {}
    open fun onPageFinished(view: WebView?, url: String?) {}
    open fun onReceivedError(view: WebView?, request: WebResourceRequest?, error: WebResourceError?) {}
    open fun onReceivedHttpError(view: WebView?, request: WebResourceRequest?, errorResponse: WebResourceResponse?) {}

    companion object {
        const val ERROR_HOST_LOOKUP = -2
        const val ERROR_CONNECT = -6
        const val ERROR_TIMEOUT = -8
    }
}

open class WebChromeClient {
    open fun onProgressChanged(view: WebView?, newProgress: Int) {}
    open fun onReceivedTitle(view: WebView?, title: String?) {}
    open fun onConsoleMessage(message: String?, lineNumber: Int, sourceID: String?) {}
    open fun getDefaultVideoPoster(): Any? = null
}

@Retention(AnnotationRetention.RUNTIME)
@Target(AnnotationTarget.FUNCTION)
annotation class JavascriptInterface

open class WebView(context: Context? = null) : View() {
    val settings: WebSettings = WebSettings()
    var webViewClient: WebViewClient = WebViewClient()
    var webChromeClient: WebChromeClient? = null

    fun loadUrl(url: String) = settleLoad(url)
    fun loadUrl(url: String, additionalHttpHeaders: MutableMap<String, String>) = settleLoad(url)
    fun loadData(data: String, mimeType: String?, encoding: String?) = settleLoad("about:blank")
    fun loadDataWithBaseURL(baseUrl: String?, data: String, mimeType: String?, encoding: String?, historyUrl: String?) =
        settleLoad(baseUrl ?: "about:blank")

    private fun settleLoad(url: String) {
        SETTLE.execute {
            runCatching { webViewClient.onPageStarted(this, url, null) }
            runCatching { webChromeClient?.onProgressChanged(this, 100) }
            runCatching { webViewClient.onPageFinished(this, url) }
        }
    }
    fun evaluateJavascript(script: String, resultCallback: ValueCallback<String>?) {
        resultCallback?.onReceiveValue("null")
    }
    fun addJavascriptInterface(obj: Any, name: String) {}
    fun removeJavascriptInterface(name: String) {}
    fun stopLoading() {}
    fun clearCache(includeDiskFiles: Boolean) {}
    fun clearFormData() {}
    fun clearHistory() {}
    fun destroy() {}
    fun onPause() {}
    fun onResume() {}

    companion object {
        private val SETTLE = Executors.newSingleThreadExecutor { r ->
            Thread(r, "sandbox-webview").apply { isDaemon = true }
        }

        @JvmStatic
        fun setWebContentsDebuggingEnabled(enabled: Boolean) {}

        @JvmStatic
        fun getWebViewClassLoader(): ClassLoader? = WebView::class.java.classLoader
    }
}
