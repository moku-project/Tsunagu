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
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.ConcurrentLinkedQueue

/**
 * Loads every playable video of an episode the way Aniyomi's EpisodeLoader does:
 * extensions-lib 16 sources expose hosters (each with its own, possibly lazily
 * resolved, videos); older ones return the videos from getVideoList(episode).
 */
object AnimeVideoLoader {
    private val logger = KotlinLogging.logger {}
    private val hosterMethods = setOf("getHosterList", "hosterListRequest", "hosterListParse")
    private val usesHostersByClass = ConcurrentHashMap<Class<*>, Boolean>()
    private const val HOSTER_TIMEOUT_MS = 45_000L
    private const val RESOLVE_TIMEOUT_MS = 30_000L
    private const val PARALLEL_RESOLVES = 4

    suspend fun loadVideos(source: AnimeHttpSource, episode: SEpisode): List<Video> {
        val errors = ConcurrentLinkedQueue<String>()
        val videos = if (usesHosters(source)) {
            val hosters = with(source) { getHosterList(episode).sortHosters() }
            coroutineScope {
                hosters.map { async(Dispatchers.IO) { videosOf(source, it, errors) } }.awaitAll()
            }.flatten()
        } else {
            with(source) { getVideoList(episode).sortVideos() }
        }
        val resolved = resolveAll(source, videos, errors)
        if (resolved.isEmpty() && errors.isNotEmpty()) {
            throw IllegalStateException("no playable video: ${errors.take(3).joinToString("; ")}")
        }
        return uniqueTitles(resolved)
    }

    /** Aniyomi's check: the extension class itself overrides a hoster method. */
    private fun usesHosters(source: AnimeHttpSource): Boolean =
        usesHostersByClass.getOrPut(source.javaClass) {
            generateSequence<Class<*>>(source.javaClass) { it.superclass }
                .takeWhile { it != AnimeHttpSource::class.java && it != ParsedAnimeHttpSource::class.java }
                .any { cls -> cls.declaredMethods.any { it.name in hosterMethods } }
        }

    private suspend fun videosOf(source: AnimeHttpSource, hoster: Hoster, errors: MutableCollection<String>): List<Video> {
        val videos = try {
            withTimeoutOrNull(HOSTER_TIMEOUT_MS) { hoster.videoList ?: source.getVideoList(hoster) }
                ?: return emptyList<Video>().also { errors += "${hoster.hosterName}: timeout" }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            logger.warn { "hoster ${hoster.hosterName} failed: ${e.javaClass.simpleName}: ${e.message}" }
            errors += "${hoster.hosterName}: ${e.message ?: e.javaClass.simpleName}"
            return emptyList()
        }
        val sorted = with(source) { videos.sortVideos() }
        if (hoster.hosterName.isBlank() || hoster.hosterName == Hoster.NO_HOSTER_LIST) return sorted
        // Prefix the hoster so the player can tell the servers apart. Fields are set in
        // place: copy() would drop the page URL that pre-lib-16 videos resolve from.
        for (video in sorted) {
            if (!video.videoTitle.contains(hoster.hosterName, ignoreCase = true)) {
                video.videoTitle = "${hoster.hosterName} - ${video.videoTitle}".trimEnd(' ', '-')
            }
        }
        return sorted
    }

    private suspend fun resolveAll(source: AnimeHttpSource, videos: List<Video>, errors: MutableCollection<String>): List<Video> {
        val gate = Semaphore(PARALLEL_RESOLVES)
        return coroutineScope {
            videos.map { async(Dispatchers.IO) { gate.withPermit { resolve(source, it, errors) } } }.awaitAll()
        }.filterNotNull()
    }

    @Suppress("DEPRECATION")
    private suspend fun resolve(source: AnimeHttpSource, video: Video, errors: MutableCollection<String>): Video? =
        try {
            withTimeoutOrNull(RESOLVE_TIMEOUT_MS) {
                val resolved = if (video.initialized) video else source.resolveVideo(video)
                when {
                    resolved == null -> null
                    // Pre-lib-16 sources leave the URL to getVideoUrl.
                    resolved.videoUrl == "null" -> resolved.apply { videoUrl = source.getVideoUrl(this); initialized = true }
                    else -> resolved.apply { initialized = true }
                }
            }?.takeIf { it.videoUrl.isNotBlank() && it.videoUrl != "null" }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            logger.warn { "resolving ${video.videoTitle} failed: ${e.javaClass.simpleName}: ${e.message}" }
            errors += "${video.videoTitle}: ${e.message ?: e.javaClass.simpleName}"
            null
        }

    /** The backend picks a stream by its label, so repeated labels would hide every copy but the first. */
    private fun uniqueTitles(videos: List<Video>): List<Video> {
        val seen = HashMap<String, Int>()
        for (video in videos) {
            val count = seen.merge(video.videoTitle, 1, Int::plus)!!
            if (count > 1) video.videoTitle = "${video.videoTitle} ($count)"
        }
        return videos
    }
}
