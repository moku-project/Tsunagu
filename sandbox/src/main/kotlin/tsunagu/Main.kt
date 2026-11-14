package tsunagu

import eu.kanade.tachiyomi.network.installConscrypt
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
import io.grpc.protobuf.services.ProtoReflectionService
import java.io.File
import java.net.InetAddress
import java.net.InetSocketAddress
import java.util.concurrent.TimeUnit
import tsunagu.grpc.ExtensionServiceImpl

fun main() {
    installConscrypt()
    val port = System.getenv("SANDBOX_PORT")?.toIntOrNull() ?: 50051
    val registry = bootstrap(
        extensionsDir = File(System.getenv("SANDBOX_EXTENSIONS_DIR") ?: "extensions"),
        storageDir = File(System.getenv("SANDBOX_STORAGE_DIR") ?: "plugin-storage"),
        novelEnabled = System.getenv("SANDBOX_ENABLE_NOVEL")?.toBooleanStrictOrNull() ?: false,
    )
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
