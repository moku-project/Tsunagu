package eu.kanade.tachiyomi.source

@Suppress("unused")
interface SourceFactory {
    fun createSources(): List<Source>
}
