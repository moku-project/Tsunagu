package tsunagu.grpc

import eu.kanade.tachiyomi.animesource.model.Hoster
import eu.kanade.tachiyomi.animesource.model.SEpisode
import eu.kanade.tachiyomi.animesource.model.Video
import eu.kanade.tachiyomi.animesource.online.AnimeHttpSource
import eu.kanade.tachiyomi.animesource.online.ParsedAnimeHttpSource
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import kotlinx.coroutines.withTimeoutOrNull

/**
 * Loads every playable video of an episode the way Aniyomi does: extensions-lib 16 sources
 * expose hosters (each with its own video list and lazily resolved videos), older ones return
 * the videos straight from getVideoList(episode).
 */
object AnimeVideoLoader {
    private val logger = KotlinLogging.logger {}
    private val hosterMethods = setOf("getHosterList", "hosterListRequest", "hosterListParse")
    private const val HOSTER_TIMEOUT_MS = 45_000L
    private const val RESOLVE_TIMEOUT_MS = 30_000L
    private const val PARALLEL_RESOLVES = 4

    /** Same check as Aniyomi's EpisodeLoader: the extension class itself overrides a hoster method. */
    fun usesHosters(source: AnimeHttpSource): Boolean {
        var current: Class<*>? = source.javaClass
        while (current != null &&
            current != AnimeHttpSource::class.java &&
            current != ParsedAnimeHttpSource::class.java
        ) {
            if (current.declaredMethods.any { it.name in hosterMethods }) return true
            current = current.superclass
        }
        return false
    }

    suspend fun loadVideos(source: AnimeHttpSource, episode: SEpisode): List<Video> =
        uniqueTitles(loadAll(source, episode))

    /** The backend picks a stream by its label, so repeated labels would hide every copy but the first. */
    private fun uniqueTitles(videos: List<Video>): List<Video> {
        val seen = HashMap<String, Int>()
        return videos.map { video ->
            val count = seen.merge(video.videoTitle, 1, Int::plus)!!
            if (count == 1) video else video.copy(videoTitle = "${video.videoTitle} ($count)")
        }
    }

    private suspend fun loadAll(source: AnimeHttpSource, episode: SEpisode): List<Video> {
        if (!usesHosters(source)) {
            val videos = with(source) { source.getVideoList(episode).sortVideos() }
            return resolveAll(source, videos, mutableListOf())
        }

        val hosters = with(source) { source.getHosterList(episode).sortHosters() }
        val errors = mutableListOf<String>()
        val perHoster = coroutineScope {
            hosters.map { hoster ->
                async(Dispatchers.IO) {
                    try {
                        withTimeoutOrNull(HOSTER_TIMEOUT_MS) { videosOf(source, hoster) }
                            ?: emptyList<Video>().also { synchronized(errors) { errors += "${hoster.hosterName}: timeout" } }
                    } catch (e: CancellationException) {
                        throw e
                    } catch (e: Throwable) {
                        logger.warn { "hoster ${hoster.hosterName} failed: ${e.javaClass.simpleName}: ${e.message}" }
                        synchronized(errors) { errors += "${hoster.hosterName}: ${e.message ?: e.javaClass.simpleName}" }
                        emptyList()
                    }
                }
            }.awaitAll()
        }
        val videos = resolveAll(source, perHoster.flatten(), errors)
        if (videos.isEmpty() && errors.isNotEmpty()) {
            throw IllegalStateException("no playable video from ${hosters.size} hoster(s): ${errors.take(3).joinToString("; ")}")
        }
        return videos
    }

    private suspend fun videosOf(source: AnimeHttpSource, hoster: Hoster): List<Video> {
        val videos = hoster.videoList ?: source.getVideoList(hoster)
        val sorted = with(source) { videos.sortVideos() }
        if (hoster.hosterName.isBlank() || hoster.hosterName == Hoster.NO_HOSTER_LIST) return sorted
        return sorted.map { video ->
            if (video.videoTitle.contains(hoster.hosterName, ignoreCase = true)) {
                video
            } else {
                video.copy(videoTitle = "${hoster.hosterName} - ${video.videoTitle}".trimEnd(' ', '-'))
            }
        }
    }

    private suspend fun resolveAll(source: AnimeHttpSource, videos: List<Video>, errors: MutableList<String>): List<Video> {
        val gate = Semaphore(PARALLEL_RESOLVES)
        return coroutineScope {
            videos.map { video ->
                async(Dispatchers.IO) { gate.withPermit { resolve(source, video, errors) } }
            }.awaitAll()
        }.filterNotNull()
    }

    @Suppress("DEPRECATION")
    private suspend fun resolve(source: AnimeHttpSource, video: Video, errors: MutableList<String>): Video? {
        return try {
            withTimeoutOrNull(RESOLVE_TIMEOUT_MS) {
                val resolved = if (video.initialized) video else source.resolveVideo(video)
                when {
                    resolved == null -> null
                    resolved.videoUrl == "null" -> resolved.copy(videoUrl = source.getVideoUrl(resolved), initialized = true)
                    else -> resolved.copy(initialized = true)
                }
            }?.takeIf { it.videoUrl.isNotBlank() && it.videoUrl != "null" }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            logger.warn { "resolving ${video.videoTitle} failed: ${e.javaClass.simpleName}: ${e.message}" }
            synchronized(errors) { errors += "${video.videoTitle}: ${e.message ?: e.javaClass.simpleName}" }
            null
        }
    }
}
