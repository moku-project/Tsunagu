package tsunagu.loader

import eu.kanade.tachiyomi.source.Source
import eu.kanade.tachiyomi.source.SourceFactory
import net.dongliu.apk.parser.ApkFile
import org.w3c.dom.Element
import org.w3c.dom.Node
import java.io.File
import java.util.zip.ZipFile
import javax.xml.parsers.DocumentBuilderFactory

data class LoadedExtension(
    val packageName: String,
    val source: Source,
    val classLoader: ClassLoader,
    val contentType: ContentType,
)

class ExtensionLoadException(message: String) : Exception(message)

object ExtensionLoader {
    private const val METADATA_SOURCE_CLASS = "tachiyomi.extension.class"

    fun load(file: File): LoadedExtension {
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

                val sourceClassMeta = getManifestMetaData(doc, METADATA_SOURCE_CLASS)
                    ?: throw ExtensionLoadException("no $METADATA_SOURCE_CLASS meta-data in manifest for $packageName")

                className = sourceClassMeta.trim().let {
                    if (it.startsWith(".")) packageName + it else it
                }
            }
            jarFile = file
        } else {
            val apk = ApkFile(file)
            packageName = apk.apkMeta.packageName

            val dbFactory = DocumentBuilderFactory.newInstance()
            val dBuilder = dbFactory.newDocumentBuilder()
            val doc = apk.manifestXml.byteInputStream().use { dBuilder.parse(it) }

            val sourceClassMeta = getManifestMetaData(doc, METADATA_SOURCE_CLASS)
                ?: throw ExtensionLoadException("no $METADATA_SOURCE_CLASS meta-data in manifest for $packageName")

            className = sourceClassMeta.trim().let {
                if (it.startsWith(".")) packageName + it else it
            }

            jarFile = Dex2JarConverter.convert(file)
        }

        val classLoader = ChildFirstURLClassLoader(arrayOf(jarFile.toURI().toURL()), javaClass.classLoader)

        val clazz = classLoader.loadClass(className)
        val instance = clazz.getDeclaredConstructor().newInstance()

        val source: Source = when (instance) {
            is Source -> instance
            is SourceFactory -> instance.createSources().firstOrNull()
                ?: throw ExtensionLoadException("$className is a SourceFactory but createSources() returned nothing")
            else -> throw ExtensionLoadException(
                "$className does not implement eu.kanade.tachiyomi.source.Source or SourceFactory " +
                    "(loaded as ${instance.javaClass.name} — check for a duplicate/stub definition " +
                    "of eu.kanade.tachiyomi.source.Source on the classpath)",
            )
        }

        val contentType = ContentTypeClassifier.classify(source.javaClass)
            ?: throw ExtensionLoadException("could not classify content type for $className")

        return LoadedExtension(packageName, source, classLoader, contentType)
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
