package tsunagu.grpc

import okhttp3.Request

// A source's image request can carry headers tied to its page, which the image URL alone loses.
internal object PageRequests {
    private const val MAX = 4096

    private val requests = object : LinkedHashMap<String, Request>(256, 0.75f, true) {
        override fun removeEldestEntry(eldest: MutableMap.MutableEntry<String, Request>) = size > MAX
    }

    private fun key(extensionId: String, url: String) = "$extensionId\u0000$url"

    @Synchronized
    fun remember(extensionId: String, request: Request): String {
        val url = request.url.toString()
        requests[key(extensionId, url)] = request
        return url
    }

    @Synchronized
    fun get(extensionId: String, url: String): Request? = requests[key(extensionId, url)]
}
