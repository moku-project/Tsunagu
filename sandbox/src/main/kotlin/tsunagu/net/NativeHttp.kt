package tsunagu.net

import okhttp3.Cookie
import okhttp3.Headers
import okhttp3.HttpUrl
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Request
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import okhttp3.internal.connection.RealCall
import okio.Buffer
import java.net.ProtocolException

/** HTTPS through the host OS (NSURLSession on iOS); the embedder registers [execute] and sets `tsunagu.nativeHttp`. */
object NativeHttp {
    val enabled: Boolean get() = System.getProperty("tsunagu.nativeHttp") == "true"

    /** Returns [status: Int, header name/value pairs: Array<String>, body: ByteArray]; throws IOException on failure. */
    @JvmStatic
    external fun execute(
        method: String,
        url: String,
        headers: Array<String>,
        body: ByteArray?,
        readTimeoutMillis: Int,
        callTimeoutMillis: Int,
    ): Array<Any?>

    /** Thrown by the native side when the OS refuses a request (e.g. App Transport Security), so OkHttp sends it instead. */
    class Refused(message: String) : java.io.IOException(message)

    /** The native call; tests swap in a fake. */
    @Volatile
    internal var send: (String, String, Array<String>, ByteArray?, Int, Int) -> Array<Any?> = ::execute

    /** Install first; it runs the client's remaining interceptors itself, then sends natively instead of over OkHttp sockets. */
    val interceptor: Interceptor = NativeTransportInterceptor()
}

// RealCall exposes the client, so later interceptors added through newBuilder() still run.
@OptIn(okhttp3.internal.OkHttpInternalApi::class)
private class NativeTransportInterceptor : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val call = chain.call() as? RealCall
        val request = chain.request()
        if (!NativeHttp.enabled || call == null || call.forWebSocket || !request.url.isHttps) {
            return chain.proceed(request)
        }
        val client = call.client
        val after = client.interceptors.dropWhile { it !== this }.drop(1)
        val steps = after + RedirectStep(client) + CookieStep(client) + client.networkInterceptors + SendStep(client)
        return Step(chain, steps, 0, request).proceed(request)
    }
}

/** A minimal chain over our own interceptor list; everything else delegates to the real call's chain. */
private class Step(
    private val outer: Interceptor.Chain,
    private val steps: List<Interceptor>,
    private val index: Int,
    private val current: Request,
) : Interceptor.Chain by outer {
    override fun request(): Request = current

    override fun connection() = null

    override fun proceed(request: Request): Response {
        if (index >= steps.size) throw IllegalStateException("native transport chain ran past its end")
        return steps[index].intercept(Step(outer, steps, index + 1, request))
    }
}

private class RedirectStep(private val client: OkHttpClient) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        var request = chain.request()
        var prior: Response? = null
        for (attempt in 0..MAX_REDIRECTS) {
            val raw = chain.proceed(request)
            val response = prior?.let { raw.newBuilder().priorResponse(it.stripBody()).build() } ?: raw
            val next = followUp(response) ?: return response
            response.close()
            prior = response
            request = next
        }
        throw ProtocolException("Too many follow-up requests: ${MAX_REDIRECTS + 1}")
    }

    private fun followUp(response: Response): Request? {
        if (!client.followRedirects) return null
        if (response.code !in listOf(300, 301, 302, 303, 307, 308)) return null
        val location = response.header("Location") ?: return null
        val url = response.request.url.resolve(location) ?: return null
        if (url.scheme != response.request.url.scheme && !client.followSslRedirects) return null
        val builder = response.request.newBuilder().url(url)
        val method = response.request.method
        if (response.code in listOf(307, 308)) {
            builder.method(method, response.request.body)
        } else if (method != "GET" && method != "HEAD") {
            builder.method("GET", null).removeHeader("Content-Type")
        }
        if (url.host != response.request.url.host) builder.removeHeader("Authorization")
        return builder.build()
    }

    private fun Response.stripBody() = newBuilder().body(ByteArray(0).toResponseBody(null)).build()

    companion object {
        const val MAX_REDIRECTS = 20
    }
}

/** What OkHttp's BridgeInterceptor would do for cookies and content type. */
private class CookieStep(private val client: OkHttpClient) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val builder = request.newBuilder()
        val cookies = client.cookieJar.loadForRequest(request.url)
        if (cookies.isNotEmpty() && request.header("Cookie") == null) {
            builder.header("Cookie", cookies.joinToString("; ") { "${it.name}=${it.value}" })
        }
        request.body?.contentType()?.let { if (request.header("Content-Type") == null) builder.header("Content-Type", it.toString()) }
        val response = chain.proceed(builder.build())
        saveCookies(request.url, response.headers)
        return response.newBuilder().request(request).build()
    }

    private fun saveCookies(url: HttpUrl, headers: Headers) {
        val cookies = Cookie.parseAll(url, headers)
        if (cookies.isNotEmpty()) client.cookieJar.saveFromResponse(url, cookies)
    }
}

private val log = io.github.oshai.kotlinlogging.KotlinLogging.logger {}
private val announced = java.util.concurrent.atomic.AtomicBoolean(false)

private class SendStep(private val client: OkHttpClient) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val body = request.body?.let { Buffer().also(it::writeTo).readByteArray() }
        val pairs = ArrayList<String>(request.headers.size * 2)
        for ((name, value) in request.headers) {
            pairs += name
            pairs += value
        }
        if (!request.url.isHttps) return fallback(request)
        if (announced.compareAndSet(false, true)) log.info { "HTTPS goes through the native transport" }
        val sent = System.currentTimeMillis()
        val result = try {
            NativeHttp.send(
                request.method, request.url.toString(), pairs.toTypedArray(), body,
                chain.readTimeoutMillis(), client.callTimeoutMillis,
            )
        } catch (_: NativeHttp.Refused) {
            return fallback(request)
        }
        val code = result[0] as Int
        @Suppress("UNCHECKED_CAST")
        val headers = Headers.headersOf(*(result[1] as Array<String>))
        val bytes = result[2] as ByteArray
        return Response.Builder()
            .request(request)
            .protocol(Protocol.HTTP_1_1)
            .code(code)
            .message("")
            .headers(headers)
            .body(bytes.toResponseBody(headers["Content-Type"]?.toMediaTypeOrNull()))
            .sentRequestAtMillis(sent)
            .receivedResponseAtMillis(System.currentTimeMillis())
            .build()
    }

    // Headers and cookies are already final, so a bare copy of the client only moves bytes.
    private fun fallback(request: Request): Response {
        val bare = client.newBuilder().apply {
            interceptors().clear()
            networkInterceptors().clear()
        }.followRedirects(false).cookieJar(okhttp3.CookieJar.NO_COOKIES).build()
        return bare.newCall(request).execute()
    }
}
