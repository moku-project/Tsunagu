package tsunagu

import io.grpc.CallOptions
import io.grpc.Channel
import io.grpc.ClientCall
import io.grpc.ClientInterceptor
import io.grpc.ForwardingClientCall
import io.grpc.ManagedChannel
import io.grpc.Metadata
import io.grpc.MethodDescriptor
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.grpc.netty.shaded.io.grpc.netty.NettyChannelBuilder
import java.nio.file.Files
import java.util.concurrent.TimeUnit
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import org.junit.jupiter.api.AfterAll
import org.koin.core.context.stopKoin
import sandbox.v1.ExtensionServiceGrpc
import sandbox.v1.Sandbox

class EmbeddedTest {
    private val token = "t".repeat(32)
    private val channels = mutableListOf<ManagedChannel>()

    @AfterTest
    fun tearDown() {
        channels.forEach { it.shutdownNow() }
        Embedded.stop()
    }

    companion object {
        // Embedded bootstraps Koin once per process; other test classes in this
        // JVM start their own.
        @JvmStatic
        @AfterAll
        fun stopKoinAfterAll() = stopKoin()
    }

    private fun stub(port: Int, bearer: String?): ExtensionServiceGrpc.ExtensionServiceBlockingStub {
        val channel = NettyChannelBuilder.forAddress("127.0.0.1", port).usePlaintext().build()
        channels += channel
        val auth = object : ClientInterceptor {
            override fun <Req, Resp> interceptCall(
                method: MethodDescriptor<Req, Resp>, options: CallOptions, next: Channel,
            ): ClientCall<Req, Resp> =
                object : ForwardingClientCall.SimpleForwardingClientCall<Req, Resp>(next.newCall(method, options)) {
                    override fun start(listener: Listener<Resp>, headers: Metadata) {
                        bearer?.let {
                            headers.put(Metadata.Key.of("authorization", Metadata.ASCII_STRING_MARSHALLER), "Bearer $it")
                        }
                        super.start(listener, headers)
                    }
                }
        }
        return ExtensionServiceGrpc.newBlockingStub(channel).withInterceptors(auth)
            .withDeadlineAfter(10, TimeUnit.SECONDS)
    }

    @Test
    fun requiresTokenAndRestarts() {
        val dir = Files.createTempDirectory("embedded").toString()
        val port = Embedded.start(dir, token, novelEnabled = false)

        stub(port, token).listLoadedExtensions(Sandbox.Empty.getDefaultInstance())
        for (bad in listOf(null, "wrong")) {
            val err = assertFailsWith<StatusRuntimeException> {
                stub(port, bad).listLoadedExtensions(Sandbox.Empty.getDefaultInstance())
            }
            assertEquals(Status.Code.UNAUTHENTICATED, err.status.code)
        }

        // Backgrounding stops the host; coming back starts it on a new port
        // without re-bootstrapping.
        Embedded.stop()
        val again = Embedded.start(dir, token, novelEnabled = false)
        stub(again, token).listLoadedExtensions(Sandbox.Empty.getDefaultInstance())
    }

    @Test
    fun rejectsShortToken() {
        assertFailsWith<IllegalArgumentException> {
            Embedded.start(Files.createTempDirectory("embedded").toString(), "short", novelEnabled = false)
        }
    }
}
