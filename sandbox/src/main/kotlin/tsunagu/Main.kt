package tsunagu

import eu.kanade.tachiyomi.network.NetworkHelper
import eu.kanade.tachiyomi.network.installConscrypt
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
import io.grpc.protobuf.services.ProtoReflectionService
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.URLConnection
import java.util.concurrent.TimeUnit
import kotlinx.serialization.json.Json
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.grpc.ExtensionServiceImpl
import tsunagu.grpc.MemoStore
import tsunagu.novel.PluginStorage
import tsunagu.registry.ExtensionRegistry
import tsunagu.source.GetSource
import java.io.File

fun main() {
    // JarURLConnection caches an open JarFile handle globally by default, outside
    // any classloader's lifecycle — on Windows that keeps extension jars locked
    // even after their classloader is closed, breaking updates/uninstalls.
    URLConnection.setDefaultUseCaches("jar", false)
    installConscrypt()
    startKoin {
        modules(
            module {
                // explicitNulls = false matters independently of ignoreUnknownKeys: a
                // nullable property with no `= null` default still throws
                // MissingFieldException when its key is absent from the JSON unless
                // this is set — several extensions (e.g. Flame Comics) rely on that.
                single { Json { ignoreUnknownKeys = true; explicitNulls = false } }
                single { NetworkHelper() }
                single { android.app.Application() }
            }
        )
    }
    val port = System.getenv("SANDBOX_PORT")?.toIntOrNull() ?: 50051
    val extensionsDir = File(System.getenv("SANDBOX_EXTENSIONS_DIR") ?: "extensions")
    val storageDir = File(System.getenv("SANDBOX_STORAGE_DIR") ?: "plugin-storage")
    val novelEnabled = System.getenv("SANDBOX_ENABLE_NOVEL")?.toBooleanStrictOrNull() ?: false
    PluginStorage.baseDir = storageDir
    MemoStore.baseDir = File(storageDir, "memo")
    android.app.Application.prefsRoot = File(storageDir, "shared-prefs")
    val registry = ExtensionRegistry(extensionsDir, novelEnabled)
    GetSource.bind(registry)
    registry.loadAll()
    val server = NettyServerBuilder
        // Loopback only: loading a jar runs its code.
        .forAddress(InetSocketAddress(InetAddress.getLoopbackAddress(), port))
        .addService(ExtensionServiceImpl(registry))
        .addService(ProtoReflectionService.newInstance())
        .permitKeepAliveTime(15, TimeUnit.SECONDS)
        .permitKeepAliveWithoutCalls(true)
        .build()
        .start()
    println("tsunagu sandbox listening on ${InetAddress.getLoopbackAddress().hostAddress}:$port")
    server.awaitTermination()
}
