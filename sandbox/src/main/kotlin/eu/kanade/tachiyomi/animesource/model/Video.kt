package eu.kanade.tachiyomi.animesource.model

import okhttp3.Headers

data class Video(
    var videoUrl: String,
    var videoTitle: String,
    var resolution: Int? = null,
    var bitrate: Int? = null,
    var headers: Headers? = null,
    var preferred: Boolean = false,
    var subtitleTracks: List<Track> = emptyList(),
    var audioTracks: List<Track> = emptyList(),
    var timestamps: List<TimeStamp> = emptyList(),
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
        uri: Any? = null,
        headers: Headers? = null,
    ) : this(url, quality, videoUrl, headers)
}

data class Track(val url: String, val lang: String)
data class TimeStamp(val start: Long, val end: Long, val name: String, val type: ChapterType)
enum class ChapterType {
    OPENING,
    ENDING,
    RECAP,
    OTHER,
}
