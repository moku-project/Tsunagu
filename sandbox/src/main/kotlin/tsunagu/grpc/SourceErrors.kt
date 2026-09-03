package tsunagu.grpc

import eu.kanade.tachiyomi.network.HttpException
import io.grpc.Status
import io.grpc.StatusRuntimeException
import java.io.IOException
import java.net.SocketTimeoutException
import java.net.UnknownHostException

/**
 * Classifies extension/source failures into stable machine codes so the backend
 * and UI can react (offer the Cloudflare solver, say "source is down", etc.)
 * instead of surfacing a raw stack trace.
 *
 * The gRPC error description is "<CODE>: <human message>"; the backend parses
 * the CODE prefix back out.
 */
object SourceErrors {

    const val CLOUDFLARE = "SOURCE_CLOUDFLARE"
    const val NOT_FOUND = "SOURCE_NOT_FOUND"
    const val UNAVAILABLE = "SOURCE_UNAVAILABLE"
    const val RATE_LIMITED = "SOURCE_RATE_LIMITED"
    const val NETWORK = "SOURCE_NETWORK"
    const val PARSE = "SOURCE_PARSE"
    const val INTERNAL = "INTERNAL"

    private fun chain(e: Throwable): Sequence<Throwable> =
        generateSequence(e as Throwable?) { it.cause }

    private fun httpCode(e: Throwable): Int? = chain(e)
        .mapNotNull { c ->
            (c as? HttpException)?.code
                ?: Regex("HTTP error (\\d{3})").find(c.message ?: "")?.groupValues?.get(1)?.toIntOrNull()
        }
        .firstOrNull()

    fun classify(e: Throwable): Pair<String, String> {
        val text = chain(e)
            .joinToString(" | ") { "${it.javaClass.simpleName}: ${it.message?.take(200) ?: ""}" }
            .lowercase()

        if ("open in webview" in text || "cloudflare" in text || "cf_clearance" in text ||
            "cf-chl" in text || "just a moment" in text
        ) {
            return CLOUDFLARE to "This source is behind Cloudflare protection."
        }

        httpCode(e)?.let { code ->
            return when {
                code == 404 || code == 410 -> NOT_FOUND to "The source returned $code (entry or endpoint missing)."
                code == 429 -> RATE_LIMITED to "The source is rate-limiting requests (429). Try again later."
                code == 403 -> CLOUDFLARE to "The source returned 403 (likely bot protection)."
                code in 500..599 -> UNAVAILABLE to "The source is temporarily unavailable (HTTP $code)."
                else -> UNAVAILABLE to "The source request failed (HTTP $code)."
            }
        }

        if (chain(e).any { it is UnknownHostException || it is SocketTimeoutException }) {
            return NETWORK to "Could not reach the source (network error)."
        }
        if (chain(e).any { it is IOException }) {
            return UNAVAILABLE to "The source request failed (${e.javaClass.simpleName})."
        }

        return if ("jsoup" in text || "nullpointerexception" in text ||
            "indexoutofbounds" in text || "no element" in text || "parse" in text
        ) {
            PARSE to "The source response could not be parsed. Its site layout may have changed."
        } else {
            INTERNAL to (e.message?.take(300) ?: e.javaClass.simpleName)
        }
    }

    fun toStatus(e: Throwable): StatusRuntimeException {
        val (code, human) = classify(e)
        val status = when (code) {
            NOT_FOUND -> Status.NOT_FOUND
            RATE_LIMITED -> Status.RESOURCE_EXHAUSTED
            CLOUDFLARE -> Status.FAILED_PRECONDITION
            UNAVAILABLE, NETWORK -> Status.UNAVAILABLE
            PARSE -> Status.DATA_LOSS
            else -> Status.INTERNAL
        }
        return StatusRuntimeException(status.withDescription("$code: $human").withCause(e))
    }

    /** Expected upstream conditions log as one WARN line; real bugs stay ERROR. */
    fun isExpected(e: Throwable): Boolean {
        val (code, _) = classify(e)
        return code != INTERNAL
    }
}
