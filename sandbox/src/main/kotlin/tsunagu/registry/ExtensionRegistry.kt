package tsunagu.registry

import tsunagu.loader.ExtensionLoader
import tsunagu.loader.LoadedExtension
import java.io.File
import java.util.concurrent.ConcurrentHashMap

class ExtensionRegistry(private val extensionsDir: File) {
    private val extensions = ConcurrentHashMap<String, LoadedExtension>()

    init {
        extensionsDir.mkdirs()
    }

    fun loadAll() {
        extensionsDir.listFiles { file -> file.extension == "apk" }?.forEach { apk ->
            runCatching { load(apk) }
        }
    }

    fun load(apkFile: File): LoadedExtension {
        val extension = ExtensionLoader.load(apkFile)
        extensions[extension.packageName] = extension
        return extension
    }

    fun get(extensionId: String): LoadedExtension? = extensions[extensionId]

    fun list(): List<LoadedExtension> = extensions.values.toList()

    fun install(sourceApk: File, extensionId: String): LoadedExtension {
        val target = File(extensionsDir, "$extensionId.apk")
        sourceApk.copyTo(target, overwrite = true)
        return load(target)
    }

    fun uninstall(extensionId: String) {
        extensions.remove(extensionId)
        File(extensionsDir, "$extensionId.apk").delete()
    }
}