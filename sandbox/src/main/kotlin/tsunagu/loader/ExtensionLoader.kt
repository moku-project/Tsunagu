package tsunagu.loader

import net.dongliu.apk.parser.ApkFile
import java.io.File
import java.lang.reflect.Modifier


data class LoadedExtension(
    val packageName: String,
    val instance: Any,
    val classLoader: ClassLoader,
    val isLegacy: Boolean,
)

class ExtensionLoadException(message: String) : Exception(message)

object ExtensionLoader {
    private const val KEI_SOURCE_CLASS = "keiyoushi.source.KeiSource"
    private const val LEGACY_HTTP_SOURCE_CLASS = "eu.kanade.tachiyomi.source.online.HttpSource"

    fun load(apkFile: File): LoadedExtension {
        val packageName = ApkFile(apkFile).apkMeta.packageName
        val jarFile = Dex2JarConverter.convert(apkFile)
        val classLoader = ChildFirstURLClassLoader(arrayOf(jarFile.toURI().toURL()), javaClass.classLoader)

        val result = findSourceClass(jarFile, classLoader)
            ?: throw ExtensionLoadException("no source class found in $packageName")

        val sourceClass = classLoader.loadClass(result.className)
        val instance = sourceClass.getDeclaredConstructor().newInstance()

        return LoadedExtension(packageName, instance, classLoader, result.isLegacy)
    }

    private data class SourceMatch(val className: String, val isLegacy: Boolean)

    private fun findSourceClass(jarFile: File, classLoader: ClassLoader): SourceMatch? {
        val failures = mutableListOf<Pair<String, Throwable>>()
        java.util.jar.JarFile(jarFile).use { jar ->
            for (entry in jar.entries()) {
                if (!entry.name.endsWith(".class") || entry.name.contains("$")) continue
                val className = entry.name.removeSuffix(".class").replace('/', '.')
                val clazz = try {
                    classLoader.loadClass(className)
                } catch (t: Throwable) {
                    failures += className to t
                    continue
                }

                if (Modifier.isAbstract(clazz.modifiers) || clazz.isInterface) continue
                if (runCatching { clazz.getDeclaredConstructor() }.isFailure) continue

                classifySource(clazz)?.let { return SourceMatch(className, it) }
            }
        }
        if (failures.isNotEmpty()) {
            val (className, cause) = failures.last()
            throw ExtensionLoadException(
                "no source class found; ${failures.size} class(es) failed to load, last: $className: ${cause.javaClass.name}: ${cause.message}"
            )
        }
        return null
    }

    private fun classifySource(clazz: Class<*>): Boolean? {
        var current: Class<*>? = clazz
        while (current != null) {
            if (current.name == KEI_SOURCE_CLASS) return false
            if (current.name == LEGACY_HTTP_SOURCE_CLASS) return true
            current = current.superclass
        }
        return null
    }

    fun isKeiSource(extension: LoadedExtension): Boolean = !extension.isLegacy
}