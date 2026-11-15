package tsunagu

import io.grpc.Metadata
import io.grpc.Server
import io.grpc.ServerCall
import io.grpc.ServerCallHandler
import io.grpc.ServerInterceptor
import io.grpc.ServerInterceptors
import io.grpc.Status
import io.grpc.netty.shaded.io.grpc.netty.NettyServerBuilder
import io.grpc.netty.shaded.io.netty.channel.nio.NioEventLoopGroup
import io.grpc.netty.shaded.io.netty.channel.socket.nio.NioServerSocketChannel
import java.io.File
import java.net.InetAddress
import java.net.InetSocketAddress
import java.security.MessageDigest
import java.util.concurrent.TimeUnit
import tsunagu.grpc.ExtensionServiceImpl
import tsunagu.registry.ExtensionRegistry

/**
 * Runs the sandbox inside a JVM the app hosts (iOS), as the extension host for
 * Go's embedded.Start. Unlike [main] it never exits the process, spawns
 * anything, or loads native code: no Conscrypt, and NIO instead of
 * epoll/kqueue transports.
 *
 * Any app on the device can reach loopback ports, so every call must carry
 * the token passed to [start].
 */
object Embedded {
    private var server: Server? = null
    private var groups: List<NioEventLoopGroup> = emptyList()
    private var registry: ExtensionRegistry? = null

    /** Starts the host on a random loopback port and returns the port. */
    @JvmStatic
    @Synchronized
    fun start(dataDirectory: String, token: String, novelEnabled: Boolean): Int {
        check(server == null) { "embedded sandbox already started" }
        require(token.length >= 32) { "token must be at least 32 characters" }
        val data = File(dataDirectory)
        System.setProperty("java.io.tmpdir", File(data, "tmp").apply { mkdirs() }.path)
        // No JIT on iOS: GraalJS must run on Truffle's interpreter.
        System.setProperty("truffle.UseFallbackRuntime", "true")
        System.setProperty("polyglot.engine.WarnInterpreterOnly", "false")
        // Truffle extracts and System.loads a native helper (truffleattach) when
        // it runs from the class path. iOS cannot bind it, and the failure
        // surfaces later as an UnsatisfiedLinkError that breaks every novel
        // plugin. Point it at nothing so Truffle takes its supported no-attach
        // path, which the interpreter-only runtime does not need.
        System.setProperty("truffle.attach.library", File(data, "no-truffleattach").path)
        System.setProperty("polyglotimpl.AttachLibraryFailureAction", "ignore")
        System.getProperty("java.home")?.let { System.setProperty("org.graalvm.launcher.home", it) }

        val reg = registry ?: bootstrap(
            extensionsDir = File(data, "extensions"),
            storageDir = File(data, "plugin-storage").apply { mkdirs() },
            novelEnabled = novelEnabled,
        ).also { registry = it }

        val boss = NioEventLoopGroup(1)
        val worker = NioEventLoopGroup(2)
        try {
            server = NettyServerBuilder
                .forAddress(InetSocketAddress(InetAddress.getByAddress(byteArrayOf(127, 0, 0, 1)), 0))
                .channelType(NioServerSocketChannel::class.java)
                .bossEventLoopGroup(boss)
                .workerEventLoopGroup(worker)
                // Covers and long-strip pages are sent whole in one message.
                .maxInboundMessageSize(64 shl 20)
                // Match main(): the Go client sends idle pings every 20 seconds.
                .permitKeepAliveTime(15, TimeUnit.SECONDS)
                .permitKeepAliveWithoutCalls(true)
                .addService(ServerInterceptors.intercept(ExtensionServiceImpl(reg), RequireToken(token)))
                .build()
                .start()
            groups = listOf(boss, worker)
            return server!!.port
        } catch (failure: Throwable) {
            boss.shutdownGracefully()
            worker.shutdownGracefully()
            throw failure
        }
    }

    /**
     * Stops listening. Loaded extensions stay in memory, so a later [start]
     * (say, after the app returns from the background) is fast.
     */
    @JvmStatic
    @Synchronized
    fun stop() {
        server?.shutdownNow()?.awaitTermination(5, TimeUnit.SECONDS)
        server = null
        groups.forEach { it.shutdownGracefully(0, 1, TimeUnit.SECONDS) }
        groups = emptyList()
    }

    internal class RequireToken(token: String) : ServerInterceptor {
        private val expected = "Bearer $token".toByteArray()
        private val key = Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER)

        override fun <ReqT, RespT> interceptCall(
            call: ServerCall<ReqT, RespT>,
            headers: Metadata,
            next: ServerCallHandler<ReqT, RespT>,
        ): ServerCall.Listener<ReqT> {
            val supplied = headers.get(key)?.toByteArray() ?: ByteArray(0)
            if (!MessageDigest.isEqual(supplied, expected)) {
                call.close(Status.UNAUTHENTICATED, Metadata())
                return object : ServerCall.Listener<ReqT>() {}
            }
            return next.startCall(call, headers)
        }
    }
}
