package tsunagu.loader

import eu.kanade.tachiyomi.animesource.online.AnimeHttpSource
import eu.kanade.tachiyomi.source.online.HttpSource
import tsunagu.source.NovelSources

enum class ContentType { MANGA, ANIME, NOVEL }

object ContentTypeClassifier {
    fun classify(clazz: Class<*>): ContentType? =
        when {
            HttpSource::class.java.isAssignableFrom(clazz) ->
                if (NovelSources.isNovel(clazz)) ContentType.NOVEL else ContentType.MANGA
            AnimeHttpSource::class.java.isAssignableFrom(clazz) -> ContentType.ANIME
            else -> null
        }
}
