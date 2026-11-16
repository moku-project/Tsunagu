package tsunagu

import eu.kanade.tachiyomi.network.NetworkHelper
import java.io.File
import java.net.URLConnection
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.json.Json
import kotlinx.serialization.protobuf.ProtoBuf
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.grpc.MemoStore
import tsunagu.novel.PluginStorage
import tsunagu.registry.ExtensionRegistry
import tsunagu.source.GetSource

/** Setup shared by the standalone sandbox ([main]) and the in-app one ([Embedded]). */
internal fun bootstrap(extensionsDir: File, storageDir: File, novelEnabled: Boolean): ExtensionRegistry {
    // Cached JarFile handles keep extension jars locked on Windows after unload, breaking updates.
    URLConnection.setDefaultUseCaches("jar", false)
    startKoin { modules(extensionModule(storageDir)) }
    PluginStorage.baseDir = storageDir
    MemoStore.baseDir = File(storageDir, "memo")
    android.app.Application.prefsRoot = File(storageDir, "shared-prefs")
    val registry = ExtensionRegistry(extensionsDir, novelEnabled)
    GetSource.bind(registry)
    registry.loadAll()
    return registry
}

/** What extensions look up through Koin; anything missing here fails an extension as it loads. */
@OptIn(ExperimentalSerializationApi::class)
internal fun extensionModule(storageDir: File) = module {
    // explicitNulls = false lets absent nullable keys decode; some extensions rely on it.
    single { Json { ignoreUnknownKeys = true; explicitNulls = false } }
    // Typed explicitly: the companion object's own type is ProtoBuf.Default, which a lookup for ProtoBuf misses.
    single<ProtoBuf> { ProtoBuf }
    single { NetworkHelper(File(storageDir, "cookies.json")) }
    single { android.app.Application() }
}
