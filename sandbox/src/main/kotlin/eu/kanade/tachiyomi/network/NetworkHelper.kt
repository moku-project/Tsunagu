package eu.kanade.tachiyomi.network

import eu.kanade.tachiyomi.network.interceptor.CloudflareInterceptor
import eu.kanade.tachiyomi.network.interceptor.UncaughtExceptionInterceptor
import eu.kanade.tachiyomi.network.interceptor.UserAgentInterceptor
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.DelicateCoroutinesApi
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.flow.onEach
import okhttp3.Cache
import okhttp3.ConnectionSpec
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import tsunagu.source.GetSource
import java.nio.file.Files
import java.util.concurrent.TimeUnit

class NetworkHelper {
    val cookieStore = PersistentCookieStore()

    private val userAgent =
        MutableStateFlow(
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
                "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        )
    val userAgentFlow = userAgent.asStateFlow()

    fun defaultUserAgentProvider(): String = userAgent.value

    init {
        @OptIn(DelicateCoroutinesApi::class)
        userAgent
            .drop(1)
            .onEach {
                GetSource.unregisterAllSources()
            }.launchIn(GlobalScope)
    }

    private val networkCacheDir =
        Files.createTempDirectory("tsunagu_network_cache").toFile().apply {
            Runtime.getRuntime().addShutdownHook(
                Thread {
                    deleteRecursively()
                },
            )
        }

    private val baseClientBuilder: OkHttpClient.Builder
        get() {
            val builder =
                OkHttpClient
                    .Builder()
                    .cookieJar(PersistentCookieJar(cookieStore))
                    .connectTimeout(30, TimeUnit.SECONDS)
                    .readTimeout(30, TimeUnit.SECONDS)
                    .callTimeout(2, TimeUnit.MINUTES)
                    .cache(
                        Cache(
                            directory = networkCacheDir,
                            maxSize = 5L * 1024 * 1024,
                        ),
                    ).dns(buildSandboxDns(System.getenv("SANDBOX_DOH") ?: "off"))

                    .connectionSpecs(
                        listOf(
                            ConnectionSpec.RESTRICTED_TLS,
                            ConnectionSpec.MODERN_TLS,
                            ConnectionSpec.COMPATIBLE_TLS,
                            ConnectionSpec.CLEARTEXT,
                        ),
                    )
                    .addInterceptor(UncaughtExceptionInterceptor())
                    .addInterceptor(UserAgentInterceptor(::defaultUserAgentProvider))

            val httpLoggingInterceptor =
                HttpLoggingInterceptor(
                    object : HttpLoggingInterceptor.Logger {
                        val logger = KotlinLogging.logger { }

                        override fun log(message: String) {
                            logger.debug { message }
                        }
                    },
                ).apply {
                    level = HttpLoggingInterceptor.Level.BASIC
                }
            builder.addNetworkInterceptor(httpLoggingInterceptor)

            builder.addInterceptor(CloudflareInterceptor(cookieStore))

            return builder
        }

    val client by lazy { baseClientBuilder.build() }

    @Deprecated("The regular client handles Cloudflare by default")
    @Suppress("UNUSED")
    val cloudflareClient by lazy { client }
}
