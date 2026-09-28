package tsunagu.util

import java.net.URI

/** Strip a URL's origin without resolving source-owned relative identifiers against baseUrl. */
fun sourceUrlWithoutDomain(value: String): String = try {
    val uri = URI(value)
    if (uri.rawAuthority == null) {
        value
    } else {
        buildString {
            append(uri.rawPath.orEmpty())
            uri.rawQuery?.let { append('?').append(it) }
            uri.rawFragment?.let { append('#').append(it) }
        }
    }
} catch (_: Exception) {
    value
}
