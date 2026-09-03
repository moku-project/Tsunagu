package tsunagu.grpc

import eu.kanade.tachiyomi.network.HttpException
import io.grpc.Status
import java.io.IOException
import java.net.UnknownHostException
import kotlin.test.Test
import kotlin.test.assertEquals

class SourceErrorsTest {

    private fun code(e: Throwable) = SourceErrors.classify(e).first

    @Test
    fun classifiesKnownFailures() {
        assertEquals(SourceErrors.CLOUDFLARE, code(IOException("Open in WebView to bypass site protection")))
        assertEquals(SourceErrors.NOT_FOUND, code(HttpException(404)))
        assertEquals(SourceErrors.NOT_FOUND, code(RuntimeException("boom", RuntimeException("HTTP error 410"))))
        assertEquals(SourceErrors.RATE_LIMITED, code(HttpException(429)))
        assertEquals(SourceErrors.UNAVAILABLE, code(HttpException(522)))
        assertEquals(SourceErrors.CLOUDFLARE, code(HttpException(403)))
        assertEquals(SourceErrors.NETWORK, code(UnknownHostException("nope.example")))
        assertEquals(SourceErrors.PARSE, code(NullPointerException("jsoup select returned null")))
        assertEquals(SourceErrors.INTERNAL, code(IllegalStateException("something odd")))
    }

    @Test
    fun statusCarriesCodePrefix() {
        val st = SourceErrors.toStatus(HttpException(404))
        assertEquals(Status.Code.NOT_FOUND, st.status.code)
        assert(st.status.description!!.startsWith("SOURCE_NOT_FOUND: "))
    }
}
