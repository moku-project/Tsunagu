package tsunagu

import eu.kanade.tachiyomi.network.NetworkHelper
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
import io.grpc.protobuf.services.ProtoReflectionService
import java.util.concurrent.TimeUnit
import kotlinx.serialization.json.Json
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.grpc.ExtensionServiceImpl
import tsunagu.registry.ExtensionRegistry
import tsunagu.source.GetSource
import java.io.File

fun main() {
    startKoin {
        modules(
            module {
                single { Json { ignoreUnknownKeys = true } }
                single { NetworkHelper() }
                single { android.app.Application() }
            }
        )
    }
    val port = System.getenv("SANDBOX_PORT")?.toIntOrNull() ?: 50051
    val extensionsDir = File(System.getenv("SANDBOX_EXTENSIONS_DIR") ?: "extensions")
    val novelEnabled = System.getenv("SANDBOX_ENABLE_NOVEL")?.toBooleanStrictOrNull() ?: false
    val registry = ExtensionRegistry(extensionsDir, novelEnabled)
    GetSource.bind(registry)
    registry.loadAll()
    val server = NettyServerBuilder
        .forPort(port)
        .addService(ExtensionServiceImpl(registry))
        .addService(ProtoReflectionService.newInstance())
        .permitKeepAliveTime(15, TimeUnit.SECONDS)
        .permitKeepAliveWithoutCalls(true)
        .build()
        .start()
    println("tsunagu sandbox listening on :$port")
    server.awaitTermination()
}