package android.util

import java.util.Base64 as JBase64

object Base64 {
    const val DEFAULT = 0
    const val NO_PADDING = 1
    const val NO_WRAP = 2
    const val CRLF = 4
    const val URL_SAFE = 8
    const val NO_CLOSE = 16

    @JvmStatic
    fun encode(input: ByteArray, flags: Int): ByteArray {
        var encoder = if (flags and URL_SAFE != 0) JBase64.getUrlEncoder() else JBase64.getEncoder()
        if (flags and NO_PADDING != 0) encoder = encoder.withoutPadding()
        return encoder.encode(input)
    }

    @JvmStatic
    fun encodeToString(input: ByteArray, flags: Int): String = String(encode(input, flags))

    @JvmStatic
    fun decode(input: ByteArray, flags: Int): ByteArray {
        val decoder = if (flags and URL_SAFE != 0) JBase64.getUrlDecoder() else JBase64.getDecoder()
        return decoder.decode(input)
    }

    @JvmStatic
    fun decode(input: String, flags: Int): ByteArray = decode(input.toByteArray(), flags)
}
