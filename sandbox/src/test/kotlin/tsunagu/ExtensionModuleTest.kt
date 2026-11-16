package tsunagu

import java.nio.file.Files
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withContext
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromByteArray
import kotlinx.serialization.encodeToByteArray
import kotlinx.serialization.json.Json
import kotlinx.serialization.protobuf.ProtoBuf
import org.koin.core.context.GlobalContext
import org.koin.core.context.startKoin
import org.koin.core.context.stopKoin

@OptIn(ExperimentalSerializationApi::class)
class ExtensionModuleTest {
    @Serializable
    data class Title(val id: Int, val name: String)

    @AfterTest
    fun tearDown() = stopKoin()

    @Test
    fun extensionsFindTheDecodersTheyAskFor() {
        startKoin { modules(extensionModule(Files.createTempDirectory("koin").toFile())) }
        val koin = GlobalContext.get()
        val proto = koin.get<ProtoBuf>()
        val title = Title(7, "Example")
        assertEquals(title, proto.decodeFromByteArray<Title>(proto.encodeToByteArray(title)))
        assertEquals(title, koin.get<Json>().decodeFromString(Title.serializer(), """{"id":7,"name":"Example","extra":1}"""))
    }

    @Test
    fun mainDispatcherRunsOffTheCallingThread() {
        val name = runBlocking { withContext(Dispatchers.Main) { Thread.currentThread().name } }
        assertTrue(name.startsWith("tsunagu-main"), name)
    }
}
