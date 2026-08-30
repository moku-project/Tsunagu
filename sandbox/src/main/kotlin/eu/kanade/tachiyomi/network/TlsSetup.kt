package eu.kanade.tachiyomi.network

import io.github.oshai.kotlinlogging.KotlinLogging
import org.conscrypt.Conscrypt
import java.security.Security

private val logger = KotlinLogging.logger {}

fun installConscrypt() {
    if (Security.getProvider("Conscrypt") != null) return
    try {
        Security.insertProviderAt(Conscrypt.newProvider(), 1)
        val v = Conscrypt.version()
        logger.info { "TLS: Conscrypt provider installed (v${v.major()}.${v.minor()}.${v.patch()})" }
    } catch (e: Throwable) {
        logger.warn(e) { "TLS: Conscrypt unavailable on this platform, using JDK default" }
    }
}
