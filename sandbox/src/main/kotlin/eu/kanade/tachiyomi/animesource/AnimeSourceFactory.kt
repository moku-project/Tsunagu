package eu.kanade.tachiyomi.animesource

@Suppress("unused")
interface AnimeSourceFactory {
    fun createSources(): List<AnimeSource>
}
