package tsunagu

import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.loader.ExtensionLoader
import tsunagu.source.FilterList
import tsunagu.source.SourceBridge
import java.io.File

fun main() = runBlocking {
    startKoin {
        modules(
            module {
                single { Json { ignoreUnknownKeys = true } }
            }
        )
    }

    val extension = ExtensionLoader.load(File("test-extension.apk"))
    println("loaded: ${extension.packageName}, isKeiSource=${ExtensionLoader.isKeiSource(extension)}")

    val bridge = SourceBridge(extension)
    val results = bridge.getSearchManga(1, "manga", FilterList())
    println("got ${results.size} manga")
    results.take(3).forEach { println("${it.title} — ${it.url}") }
}