package eu.kanade.tachiyomi.network

import io.github.oshai.kotlinlogging.KotlinLogging
import okhttp3.Dns
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.dnsoverhttps.DnsOverHttps
import java.net.InetAddress
import java.net.UnknownHostException

private val logger = KotlinLogging.logger {}

fun buildSandboxDns(mode: String): Dns {
    if (mode.equals("off", ignoreCase = true) || mode.equals("false", ignoreCase = true)) {
        return Dns.SYSTEM
    }

    val dohUrl = mode.takeIf { it.startsWith("http") } ?: "https://cloudflare-dns.com/dns-query"
    val bootstrap =
        OkHttpClient.Builder()
            .build()
    val doh =
        try {
            DnsOverHttps.Builder()
                .client(bootstrap)
                .url(dohUrl.toHttpUrl())

                .bootstrapDnsHosts(
                    InetAddress.getByName("1.1.1.1"),
                    InetAddress.getByName("1.0.0.1"),
                    InetAddress.getByName("2606:4700:4700::1111"),
                    InetAddress.getByName("2606:4700:4700::1001"),
                )
                .build()
        } catch (e: Exception) {
            logger.warn(e) { "DoH init failed, using system DNS" }
            return Dns.SYSTEM
        }

    logger.info { "sandbox DNS: DoH via $dohUrl (system-DNS fallback)" }
    return FallbackDns(doh, Dns.SYSTEM)
}

private class FallbackDns(
    private val primary: Dns,
    private val fallback: Dns,
) : Dns {
    override fun lookup(hostname: String): List<InetAddress> =
        try {
            primary.lookup(hostname).ifEmpty { fallback.lookup(hostname) }
        } catch (e: UnknownHostException) {
            try {
                fallback.lookup(hostname)
            } catch (_: UnknownHostException) {
                throw e
            }
        }
}
