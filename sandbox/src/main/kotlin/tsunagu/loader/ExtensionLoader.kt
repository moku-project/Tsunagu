package tsunagu.loader

import eu.kanade.tachiyomi.animesource.AnimeSource
import eu.kanade.tachiyomi.animesource.AnimeSourceFactory
import eu.kanade.tachiyomi.source.Source
import eu.kanade.tachiyomi.source.SourceFactory
import net.dongliu.apk.parser.ApkFile
import org.w3c.dom.Element
import org.w3c.dom.Node
import java.io.File
import java.util.zip.ZipFile
import tsunagu.novel.NovelPluginLoader
import javax.xml.parsers.DocumentBuilderFactory

data class LoadedExtension(
    val packageName: String,
    val source: Any,
    val classLoader: ClassLoader,
    val contentType: ContentType,
)

class ExtensionLoadException(message: String) : Exception(message)

// SourceFactory extensions (e.g. MangaDex) register one Source per language; picking index 0
// blindly locks every request to whatever language sorts first alphabetically (e.g. "af").
private fun List<Source>.pickPreferredLang(): Source? = find { it.lang == "en" } ?: firstOrNull()
private fun List<AnimeSource>.pickPreferredLang(): AnimeSource? = find { it.lang == "en" } ?: firstOrNull()

object ExtensionLoader {
    private val SOURCE_CLASS_KEYS = listOf(
        "tachiyomi.extension.class",
        "tachiyomi.animeextension.class",
        "tachiyomi.novelextension.class", // Tsundoku-format novel extensions
    )

    fun peekPackageName(file: File): String {
        if (file.extension == "js") {
            // LNReader plugins declare their id in code, so the only way to read it is to load them.
            val plugin = NovelPluginLoader.load(file)
            (plugin.source as? AutoCloseable)?.close()
            return plugin.packageName
        }
        if (file.extension == "jar") {
            ZipFile(file).use { zip ->
                val manifestEntry = zip.getEntry("AndroidManifest.xml")
                    ?: throw ExtensionLoadException("no AndroidManifest.xml inside jar ${file.name}")
                val dbFactory = DocumentBuilderFactory.newInstance()
                val dBuilder = dbFactory.newDocumentBuilder()
                val doc = zip.getInputStream(manifestEntry).use { dBuilder.parse(it) }
                return doc.documentElement.getAttribute("package")
            }
        }
        val apk = ApkFile(file)
        return apk.apkMeta.packageName
    }

    fun load(file: File): LoadedExtension {
        if (file.extension == "js") {
            return NovelPluginLoader.load(file)
        }

        val className: String
        val packageName: String
        val jarFile: File

        if (file.extension == "jar") {
            ZipFile(file).use { zip ->
                val manifestEntry = zip.getEntry("AndroidManifest.xml")
                    ?: throw ExtensionLoadException("no AndroidManifest.xml inside jar ${file.name}")
                val dbFactory = DocumentBuilderFactory.newInstance()
                val dBuilder = dbFactory.newDocumentBuilder()
                val doc = zip.getInputStream(manifestEntry).use { dBuilder.parse(it) }

                packageName = doc.documentElement.getAttribute("package")

                className = sourceClassName(doc, packageName)
            }
            jarFile = file
        } else {
            val apk = ApkFile(file)
            packageName = apk.apkMeta.packageName

            val dbFactory = DocumentBuilderFactory.newInstance()
            val dBuilder = dbFactory.newDocumentBuilder()
            val doc = apk.manifestXml.byteInputStream().use { dBuilder.parse(it) }

            className = sourceClassName(doc, packageName)

            jarFile = Dex2JarConverter.convert(file)
        }

        val classLoader = ChildFirstURLClassLoader(arrayOf(jarFile.toURI().toURL()), javaClass.classLoader)

        val clazz = classLoader.loadClass(className)
        val instance = clazz.getDeclaredConstructor().newInstance()

        val source: Any = when (instance) {
            is Source -> instance
            is SourceFactory -> instance.createSources().pickPreferredLang()
                ?: throw ExtensionLoadException("$className is a SourceFactory but createSources() returned nothing")
            is AnimeSource -> instance
            is AnimeSourceFactory -> instance.createSources().pickPreferredLang()
                ?: throw ExtensionLoadException("$className is an AnimeSourceFactory but createSources() returned nothing")
            else -> throw ExtensionLoadException(
                "$className does not implement eu.kanade.tachiyomi.source.Source, SourceFactory, " +
                    "eu.kanade.tachiyomi.animesource.AnimeSource, or AnimeSourceFactory " +
                    "(loaded as ${instance.javaClass.name} — check for a duplicate/stub definition " +
                    "of eu.kanade.tachiyomi.source.Source on the classpath)",
            )
        }

        val contentType = ContentTypeClassifier.classify(source.javaClass)
            ?: throw ExtensionLoadException("could not classify content type for $className")

        return LoadedExtension(packageName, source, classLoader, contentType)
    }

    private fun sourceClassName(doc: org.w3c.dom.Document, packageName: String): String {
        val meta = SOURCE_CLASS_KEYS.firstNotNullOfOrNull { getManifestMetaData(doc, it) }
            ?: throw ExtensionLoadException("no source class meta-data (${SOURCE_CLASS_KEYS.joinToString()}) in manifest for $packageName")
        return meta.trim().let { if (it.startsWith(".")) packageName + it else it }
    }

    private fun getManifestMetaData(doc: org.w3c.dom.Document, key: String): String? {
        val appTag = doc.getElementsByTagName("application").item(0) ?: return null

        val children = appTag.childNodes
        for (i in 0 until children.length) {
            val node = children.item(i)
            if (node.nodeType != Node.ELEMENT_NODE) continue
            val element = node as Element
            if (element.tagName != "meta-data") continue

            val name = element.attributes.getNamedItem("android:name")?.nodeValue
            val value = element.attributes.getNamedItem("android:value")?.nodeValue
            if (name == key) return value
        }
        return null
    }

    private fun getManifestMetaData(apk: ApkFile, key: String): String? {
        val dbFactory = DocumentBuilderFactory.newInstance()
        val dBuilder = dbFactory.newDocumentBuilder()
        val doc = apk.manifestXml.byteInputStream().use { dBuilder.parse(it) }
        return getManifestMetaData(doc, key)
    }
}
