package tsunagu

import eu.kanade.tachiyomi.network.NetworkHelper
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
import io.grpc.protobuf.services.ProtoReflectionService
import kotlinx.serialization.json.Json
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.grpc.ExtensionServiceImpl
import tsunagu.registry.ExtensionRegistry
import tsunagu.repository.RepositoryRegistry
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
    val registry = ExtensionRegistry(extensionsDir)
    GetSource.bind(registry) // lets NetworkHelper's user-agent-change hook reset loaded sources
    registry.loadAll()
    val repositoryRegistry = RepositoryRegistry()
    val server = NettyServerBuilder
        .forPort(port)
        .addService(ExtensionServiceImpl(registry, repositoryRegistry))
        .addService(ProtoReflectionService.newInstance())
        .build()
        .start()
    println("tsunagu sandbox listening on :$port")
    server.awaitTermination()
}
