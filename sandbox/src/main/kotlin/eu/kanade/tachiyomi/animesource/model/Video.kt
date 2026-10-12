// Adapted from Aniyomi (https://github.com/aniyomiorg/aniyomi), licensed under Apache License 2.0.
// See /NOTICE.md and /THIRD_PARTY_LICENSES/Apache-2.0.txt for full attribution.

package eu.kanade.tachiyomi.animesource.model

import android.net.Uri
import okhttp3.Headers

// The primary constructor (and so the generated copy) must keep the extensions-lib 16 shape:
// lib 16 extensions are compiled against it and call it directly.
data class Video(
    var videoUrl: String = "",
    var videoTitle: String = "",
    var resolution: Int? = null,
    var bitrate: Int? = null,
    var headers: Headers? = null,
    var preferred: Boolean = false,
    var subtitleTracks: List<Track> = emptyList(),
    var audioTracks: List<Track> = emptyList(),
    var timestamps: List<TimeStamp> = emptyList(),
    var mpvArgs: List<Pair<String, String>> = emptyList(),
    var ffmpegStreamArgs: List<Pair<String, String>> = emptyList(),
    var ffmpegVideoArgs: List<Pair<String, String>> = emptyList(),
    var internalData: String = "",
    var initialized: Boolean = false,
) {
    @Deprecated("Use videoTitle instead", ReplaceWith("videoTitle"))
    var quality: String
        get() = videoTitle
        set(value) { videoTitle = value }

    val url: String
        get() = videoPageUrl

    private var videoPageUrl: String = ""

    constructor(
        url: String,
        quality: String,
        videoUrl: String?,
        headers: Headers? = null,
        subtitleTracks: List<Track> = emptyList(),
        audioTracks: List<Track> = emptyList(),
    ) : this(
        videoTitle = quality,
        videoUrl = videoUrl ?: "null",
        headers = headers,
        subtitleTracks = subtitleTracks,
        audioTracks = audioTracks,
    ) {
        this.videoPageUrl = url
    }

    @Suppress("UNUSED_PARAMETER")
    constructor(
        url: String,
        quality: String,
        videoUrl: String?,
        uri: Uri? = null,
        headers: Headers? = null,
    ) : this(url, quality, videoUrl, headers)

    @Deprecated("Only for extensions-lib 14 binaries", level = DeprecationLevel.HIDDEN)
    fun copy(
        url: String,
        quality: String,
        videoUrl: String?,
        headers: Headers? = this.headers,
        subtitleTracks: List<Track> = this.subtitleTracks,
        audioTracks: List<Track> = this.audioTracks,
    ): Video = Video(url, quality, videoUrl, headers, subtitleTracks, audioTracks)
}

data class Track(val url: String, val lang: String)

/** [start] and [end] are in seconds, as in extensions-lib 16. */
data class TimeStamp(
    val start: Double,
    val end: Double,
    val name: String,
    val type: ChapterType = ChapterType.Other,
)

enum class ChapterType {
    Opening,
    Ending,
    Recap,
    MixedOp,
    Other,
}
