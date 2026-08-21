package tsunagu.registry

import okhttp3.OkHttpClient
import okhttp3.Request
import tsunagu.loader.ExtensionLoader
import tsunagu.loader.LoadedExtension
import java.io.File
import java.nio.file.Files
import java.util.concurrent.ConcurrentHashMap

class ExtensionDownloadException(message: String) : Exception(message)
class InvalidExtensionIdException(message: String) : Exception(message)

class ExtensionRegistry(private val extensionsDir: File) {
    private val extensions = ConcurrentHashMap<String, LoadedExtension>()
    private val client = OkHttpClient()

    init {
        extensionsDir.mkdirs()
    }

    private fun requireSafeExtensionId(extensionId: String) {
        val safe = Regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
        if (!safe.matches(extensionId)) {
            throw InvalidExtensionIdException("unsafe extension id: $extensionId")
        }
    }

    private fun targetFile(extensionId: String, ext: String): File {
        requireSafeExtensionId(extensionId)
        val target = File(extensionsDir, "$extensionId.$ext").canonicalFile
        if (!target.parentFile.canonicalFile.equals(extensionsDir.canonicalFile)) {
            throw InvalidExtensionIdException("resolved path escapes extensions dir: $extensionId")
        }
        return target
    }

    fun loadAll() {
        extensionsDir.listFiles { file -> file.extension == "apk" || file.extension == "jar" || file.extension == "js" }?.forEach { file ->
            runCatching { load(file) }
                .onFailure { println("FAILED to load $file: ${it.stackTraceToString()}") }
        }
    }

    fun load(file: File): LoadedExtension {
        val extension = ExtensionLoader.load(file)
        extensions[extension.packageName] = extension
        return extension
    }

    fun get(extensionId: String): LoadedExtension? = extensions[extensionId]

    fun list(): List<LoadedExtension> = extensions.values.toList()

    fun install(sourceFile: File, extensionId: String): LoadedExtension {
        val ext = sourceFile.extension.ifBlank { "apk" }
        val target = targetFile(extensionId, ext)
        sourceFile.copyTo(target, overwrite = true)
        return load(target)
    }

    fun installFromUrl(apkUrl: String?, jarUrl: String?, jsUrl: String?, extensionId: String): LoadedExtension {
        val (url, ext) = when {
            jsUrl != null -> jsUrl to "js"
            jarUrl != null -> jarUrl to "jar"
            apkUrl != null -> apkUrl to "apk"
            else -> throw ExtensionDownloadException("no download url provided for $extensionId")
        }

        val request = Request.Builder().url(url).build()
        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            response.close()
            throw ExtensionDownloadException("failed to download $url: HTTP ${response.code}")
        }
        val bytes = response.body.bytes()

        val target = targetFile(extensionId, ext)
        Files.write(target.toPath(), bytes)
        return load(target)
    }

    fun uninstall(extensionId: String) {
        requireSafeExtensionId(extensionId)
        extensions.remove(extensionId)
        File(extensionsDir, "$extensionId.apk").delete()
        File(extensionsDir, "$extensionId.jar").delete()
        File(extensionsDir, "$extensionId.js").delete()
    }

    fun invalidateAll() {
        extensions.clear()
        loadAll()
    }
}
