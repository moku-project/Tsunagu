package tsunagu.loader

import eu.kanade.tachiyomi.source.online.HttpSource

enum class ContentType { MANGA, ANIME, NOVEL }

object ContentTypeClassifier {
    fun classify(clazz: Class<*>): ContentType? =
        if (HttpSource::class.java.isAssignableFrom(clazz)) ContentType.MANGA else null

    fun fromPackageName(packageName: String): ContentType =
        when {
            packageName.startsWith("eu.kanade.tachiyomi.animeextension.") -> ContentType.ANIME
            packageName.startsWith("eu.kanade.tachiyomi.extension.") -> ContentType.MANGA
            else -> ContentType.MANGA
        }
}
