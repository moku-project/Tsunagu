package tsunagu

import eu.kanade.tachiyomi.network.NetworkHelper
import java.io.File
import java.net.URLConnection
import kotlinx.serialization.json.Json
import org.koin.core.context.startKoin
import org.koin.dsl.module
import tsunagu.grpc.MemoStore
import tsunagu.novel.PluginStorage
import tsunagu.registry.ExtensionRegistry
import tsunagu.source.GetSource

/** Setup shared by the standalone sandbox ([main]) and the in-app one ([Embedded]). */
internal fun bootstrap(extensionsDir: File, storageDir: File, novelEnabled: Boolean): ExtensionRegistry {
    // JarURLConnection caches an open JarFile handle globally by default, outside
    // any classloader's lifecycle — on Windows that keeps extension jars locked
    // even after their classloader is closed, breaking updates/uninstalls.
    URLConnection.setDefaultUseCaches("jar", false)
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
    PluginStorage.baseDir = storageDir
    MemoStore.baseDir = File(storageDir, "memo")
    android.app.Application.prefsRoot = File(storageDir, "shared-prefs")
    val registry = ExtensionRegistry(extensionsDir, novelEnabled)
    GetSource.bind(registry)
    registry.loadAll()
    return registry
}
