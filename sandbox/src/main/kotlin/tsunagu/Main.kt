package tsunagu

import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
import io.grpc.protobuf.services.ProtoReflectionService
import kotlinx.serialization.json.Json
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.grpc.ExtensionServiceImpl
import tsunagu.registry.ExtensionRegistry
import java.io.File

fun main() {
    startKoin {
        modules(
            module {
                single { Json { ignoreUnknownKeys = true } }
            }
        )
    }

    val port = System.getenv("SANDBOX_PORT")?.toIntOrNull() ?: 50051
    val extensionsDir = File(System.getenv("SANDBOX_EXTENSIONS_DIR") ?: "extensions")

    val registry = ExtensionRegistry(extensionsDir)
    registry.loadAll()

    val server = NettyServerBuilder
        .forPort(port)
        .addService(ExtensionServiceImpl(registry))
        .addService(ProtoReflectionService.newInstance())
        .build()
        .start()

    println("tsunagu sandbox listening on :$port")
    server.awaitTermination()
}
