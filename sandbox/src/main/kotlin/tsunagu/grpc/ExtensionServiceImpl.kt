package tsunagu.grpc

import eu.kanade.tachiyomi.animesource.model.SAnime
import eu.kanade.tachiyomi.animesource.model.SEpisode
import eu.kanade.tachiyomi.animesource.model.Video
import eu.kanade.tachiyomi.animesource.online.AnimeHttpSource
import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga
import eu.kanade.tachiyomi.source.online.HttpSource
import eu.kanade.tachiyomi.network.GET
import eu.kanade.tachiyomi.network.awaitSuccess
import io.github.oshai.kotlinlogging.KotlinLogging
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.grpc.stub.StreamObserver
import kotlinx.coroutines.runBlocking
import sandbox.v1.ExtensionServiceGrpc
import sandbox.v1.Sandbox
import tsunagu.loader.ContentType
import tsunagu.loader.ExtensionLoader
import tsunagu.loader.LoadedExtension
import tsunagu.novel.ChapterItem
import tsunagu.novel.NovelPlugin
import tsunagu.novel.SourceNovel
import tsunagu.grpc.AnimeFilterSerde
import tsunagu.util.awaitSingle
import tsunagu.grpc.FilterSerde.applyTo
import tsunagu.grpc.FilterSerde.toProto
import tsunagu.registry.ExtensionRegistry
import java.io.File

class ExtensionServiceImpl(
    private val registry: ExtensionRegistry,
) : ExtensionServiceGrpc.ExtensionServiceImplBase() {

    private val logger = KotlinLogging.logger {}

    override fun loadExtensions(
        request: Sandbox.LoadExtensionsRequest,
        responseObserver: StreamObserver<Sandbox.ExtensionList>,
    ) {
        try {
            val builder = Sandbox.ExtensionList.newBuilder()
            request.extensionsList.forEach { toLoad ->
                val loaded = registry.install(File(toLoad.jarPath), toLoad.extensionId)
                builder.addExtensions(toExtensionProto(loaded))
            }
            responseObserver.onNext(builder.build())
            responseObserver.onCompleted()
        } catch (e: Exception) {
            logger.error(e) { "load extensions failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun listLoadedExtensions(
        request: Sandbox.Empty,
        responseObserver: StreamObserver<Sandbox.ExtensionList>,
    ) {
        val builder = Sandbox.ExtensionList.newBuilder()
        registry.list().forEach { ext -> builder.addExtensions(toExtensionProto(ext)) }
        responseObserver.onNext(builder.build())
        responseObserver.onCompleted()
    }

    override fun unloadExtension(
        request: Sandbox.ExtensionRequest,
        responseObserver: StreamObserver<Sandbox.Empty>,
    ) {
        registry.uninstall(request.extensionId)
        responseObserver.onNext(Sandbox.Empty.getDefaultInstance())
        responseObserver.onCompleted()
    }

    override fun search(
        request: Sandbox.SearchRequest,
        responseObserver: StreamObserver<Sandbox.SearchResponse>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        try {
            val builder = Sandbox.SearchResponse.newBuilder()
            when (val source = extension.source) {
                is HttpSource -> {
                    val filters = source.getFilterList()
                    request.filtersList.applyTo(filters)
                    val page = runBlocking { source.getSearchManga(request.page, request.query, filters) }
                    builder.setHasNextPage(page.hasNextPage)
                    page.mangas.forEach { manga -> builder.addResults(toEntrySummary(manga)) }
                }
                is AnimeHttpSource -> {
                    val filters = source.getFilterList()
                    with(AnimeFilterSerde) { request.filtersList.applyTo(filters) }
                    val page = runBlocking { source.getSearchAnime(request.page, request.query, filters) }
                    builder.setHasNextPage(page.hasNextPage)
                    page.animes.forEach { anime -> builder.addResults(toEntrySummaryAnime(anime)) }
                }
                is NovelPlugin -> {
                    val results = source.searchNovels(request.query, request.page)
                    builder.setHasNextPage(results.isNotEmpty())
                    results.forEach { novel ->
                        builder.addResults(
                            Sandbox.EntrySummary.newBuilder()
                                .setSourceEntryId(novel.path)
                                .setTitle(novel.name)
                                .setCoverUrl(novel.cover ?: "")
                                .build(),
                        )
                    }
                }
                else -> throw IllegalStateException("extension ${request.extensionId} is not searchable")
            }
            responseObserver.onNext(builder.build())
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun getFilterList(
        request: Sandbox.GetFilterListRequest,
        responseObserver: StreamObserver<Sandbox.GetFilterListResponse>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        try {
            val nodes = when (val source = extension.source) {
                is HttpSource -> source.getFilterList().toProto()
                is AnimeHttpSource -> with(AnimeFilterSerde) { source.getFilterList().toProto() }
                else -> emptyList()
            }
            val response = Sandbox.GetFilterListResponse.newBuilder().addAllFilters(nodes).build()
            responseObserver.onNext(response)
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    @Suppress("DEPRECATION")
    override fun getPopularManga(
        request: Sandbox.BrowseRequest,
        responseObserver: StreamObserver<Sandbox.SearchResponse>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        try {
            val builder = Sandbox.SearchResponse.newBuilder()
            when (val source = extension.source) {
                is HttpSource -> {
                    val page = runBlocking { source.fetchPopularManga(request.page).awaitSingle() }
                    builder.setHasNextPage(page.hasNextPage)
                    page.mangas.forEach { manga -> builder.addResults(toEntrySummary(manga)) }
                }
                is AnimeHttpSource -> {
                    val page = runBlocking { source.fetchPopularAnime(request.page).awaitSingle() }
                    builder.setHasNextPage(page.hasNextPage)
                    page.animes.forEach { anime -> builder.addResults(toEntrySummaryAnime(anime)) }
                }
                is NovelPlugin -> {
                    val results = source.popularNovels(request.page)
                    builder.setHasNextPage(results.isNotEmpty())
                    results.forEach { novel ->
                        builder.addResults(
                            Sandbox.EntrySummary.newBuilder()
                                .setSourceEntryId(novel.path)
                                .setTitle(novel.name)
                                .setCoverUrl(novel.cover ?: "")
                                .build(),
                        )
                    }
                }
                else -> throw IllegalStateException("extension ${request.extensionId} does not support popular listing")
            }
            responseObserver.onNext(builder.build())
            responseObserver.onCompleted()
        } catch (e: UnsupportedOperationException) {

            logger.info { "extension ${request.extensionId} has no popular listing, returning empty page" }
            responseObserver.onNext(Sandbox.SearchResponse.newBuilder().setHasNextPage(false).build())
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    @Suppress("DEPRECATION")
    override fun getLatestUpdates(
        request: Sandbox.BrowseRequest,
        responseObserver: StreamObserver<Sandbox.SearchResponse>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        try {
            val builder = Sandbox.SearchResponse.newBuilder()
            when (val source = extension.source) {
                is HttpSource -> {
                    val page = runBlocking { source.fetchLatestUpdates(request.page).awaitSingle() }
                    builder.setHasNextPage(page.hasNextPage)
                    page.mangas.forEach { manga -> builder.addResults(toEntrySummary(manga)) }
                }
                is AnimeHttpSource -> {
                    val page = runBlocking { source.fetchLatestUpdates(request.page).awaitSingle() }
                    builder.setHasNextPage(page.hasNextPage)
                    page.animes.forEach { anime -> builder.addResults(toEntrySummaryAnime(anime)) }
                }

                is NovelPlugin -> builder.setHasNextPage(false)
                else -> throw IllegalStateException("extension ${request.extensionId} does not support latest updates")
            }
            responseObserver.onNext(builder.build())
            responseObserver.onCompleted()
        } catch (e: UnsupportedOperationException) {

            logger.info { "extension ${request.extensionId} has no latest-updates listing, returning empty page" }
            responseObserver.onNext(Sandbox.SearchResponse.newBuilder().setHasNextPage(false).build())
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun getDetails(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.EntryDetails>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        try {
            val details = when (extension.contentType) {
                ContentType.NOVEL -> {
                    val plugin = extension.source as NovelPlugin
                    toEntryDetailsNovel(plugin.parseNovel(request.sourceEntryId))
                }
                ContentType.ANIME -> {
                    val source = extension.source as AnimeHttpSource
                    val stub = SAnime.create().apply { url = request.sourceEntryId; title = "" }

                    val details = runBlocking { source.getAnimeDetails(stub) }
                    stub.mergeDetailsFrom(details)
                    toEntryDetailsAnime(stub)
                }
                ContentType.MANGA -> {
                    val source = extension.source as HttpSource
                    val stub = SManga.create().apply { url = request.sourceEntryId; title = "" }
                    val update = runBlocking {
                        source.getMangaUpdate(stub, emptyList(), fetchDetails = true, fetchChapters = false)
                    }
                    toEntryDetails(update.manga)
                }
            }
            responseObserver.onNext(details)
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun getChapters(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.ChapterList>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        try {
            val list = when (extension.contentType) {
                ContentType.NOVEL -> {
                    val plugin = extension.source as NovelPlugin
                    val novel = plugin.parseNovel(request.sourceEntryId)
                    val builder = Sandbox.ChapterList.newBuilder()
                    novel.chapters.forEach { chapter -> builder.addChapters(toChapterSummaryNovel(chapter)) }
                    builder.build()
                }
                else -> {
                    val source = extension.source as HttpSource
                    val stub = SManga.create().apply { url = request.sourceEntryId; title = "" }
                    val update = runBlocking {
                        source.getMangaUpdate(stub, emptyList(), fetchDetails = false, fetchChapters = true)
                    }
                    val builder = Sandbox.ChapterList.newBuilder()
                    update.chapters.forEach { chapter -> builder.addChapters(toChapterSummary(chapter)) }
                    builder.build()
                }
            }
            responseObserver.onNext(list)
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun getPages(
        request: Sandbox.ChapterRequest,
        responseObserver: StreamObserver<Sandbox.PageList>,
    ) {
        handle(responseObserver, request.extensionId) { source ->
            val chapter: SChapter = SChapter.create().apply { url = request.sourceChapterId; name = "" }
            val pages = runBlocking {
                source.getPageList(chapter).map { page ->
                    page.imageUrl ?: source.getImageUrl(page)
                }
            }
            val builder = Sandbox.PageList.newBuilder()
            pages.forEach { builder.addPageUrls(it) }
            builder.build()
        }
    }

    override fun getChapterText(
        request: Sandbox.ChapterRequest,
        responseObserver: StreamObserver<Sandbox.TextContent>,
    ) {
        val extension = registry.get(request.extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(request.extensionId))
            return
        }
        if (extension.contentType != ContentType.NOVEL) {
            responseObserver.onError(unimplemented())
            return
        }
        try {
            val plugin = extension.source as NovelPlugin
            val text = plugin.parseChapter(request.sourceChapterId)
            responseObserver.onNext(
                Sandbox.TextContent.newBuilder().setContent(text).setFormat("html").build(),
            )
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun getEpisodes(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.EpisodeList>,
    ) {
        handleAnime(responseObserver, request.extensionId) { source ->
            val stub = SAnime.create().apply { url = request.sourceEntryId; title = "" }
            val episodes = runBlocking { source.getEpisodeList(stub) }
            val builder = Sandbox.EpisodeList.newBuilder()
            episodes.forEach { episode -> builder.addEpisodes(toEpisodeSummary(episode)) }
            builder.build()
        }
    }

    override fun getVideoStream(
        request: Sandbox.EpisodeRequest,
        responseObserver: StreamObserver<Sandbox.StreamInfo>,
    ) {
        handleAnime(responseObserver, request.extensionId) { source ->
            val episode: SEpisode = SEpisode.create().apply { url = request.sourceEpisodeId; name = "" }
            val videos = runBlocking { source.getVideoList(episode) }
            if (videos.isEmpty()) {
                throw IllegalStateException("no videos returned for episode ${request.sourceEpisodeId}")
            }
            val preferred = videos.firstOrNull { it.preferred }
                ?: videos.firstOrNull { it.videoUrl.contains("localhost") || it.videoUrl.contains("127.0.0.1") }
                ?: videos.first()
            logger.info {
                "getVideoStream ${request.sourceEpisodeId}: ${videos.size} source(s), " +
                    "subs=${videos.sumOf { it.subtitleTracks.size }} " +
                    "audio=${videos.sumOf { it.audioTracks.size }} " +
                    "timestamps=${preferred.timestamps.size}"
            }
            toStreamInfo(preferred, videos)
        }
    }

    override fun getImageBytes(
        request: Sandbox.ImageRequest,
        responseObserver: StreamObserver<Sandbox.ImageData>,
    ) {
        handle(responseObserver, request.extensionId) { source ->
            val response = runBlocking {
                source.client.newCall(GET(request.imageUrl, source.headers)).awaitSuccess()
            }
            response.use { resp ->
                val bytes = resp.body.bytes()
                val contentType = resp.header("Content-Type") ?: "image/jpeg"
                Sandbox.ImageData.newBuilder()
                    .setData(com.google.protobuf.ByteString.copyFrom(bytes))
                    .setContentType(contentType)
                    .build()
            }
        }
    }

    private fun <T> handle(
        responseObserver: StreamObserver<T>,
        extensionId: String,
        block: (HttpSource) -> T,
    ) {
        val extension = registry.get(extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(extensionId))
            return
        }
        try {
            val httpSource = extension.source as? HttpSource
                ?: return responseObserver.onError(internal(IllegalStateException("extension $extensionId is not an HttpSource")))
            responseObserver.onNext(block(httpSource))
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    private fun <T> handleAnime(
        responseObserver: StreamObserver<T>,
        extensionId: String,
        block: (AnimeHttpSource) -> T,
    ) {
        val extension = registry.get(extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(extensionId))
            return
        }
        try {
            val animeSource = extension.source as? AnimeHttpSource
                ?: return responseObserver.onError(internal(IllegalStateException("extension $extensionId is not an AnimeHttpSource")))
            responseObserver.onNext(block(animeSource))
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    private fun toContentTypeProto(contentType: ContentType): Sandbox.ContentType =
        when (contentType) {
            ContentType.MANGA -> Sandbox.ContentType.MANGA
            ContentType.ANIME -> Sandbox.ContentType.ANIME
            ContentType.NOVEL -> Sandbox.ContentType.NOVEL
        }

    private fun extensionSupportsLatest(source: Any): Boolean =
        when (source) {
            is eu.kanade.tachiyomi.source.Source -> source.supportsLatest
            is eu.kanade.tachiyomi.animesource.AnimeSource -> source.supportsLatest
            else -> false
        }

    private fun toExtensionProto(ext: LoadedExtension): Sandbox.Extension =
        Sandbox.Extension.newBuilder()
            .setId(ext.packageName)
            .setName(ext.packageName)
            .setContentType(toContentTypeProto(ext.contentType))
            .setLang(extensionLang(ext.source))
            .setSupportsLatest(extensionSupportsLatest(ext.source))
            .build()

    private fun toEntrySummary(manga: SManga): Sandbox.EntrySummary =
        Sandbox.EntrySummary.newBuilder()
            .setSourceEntryId(manga.url)
            .setTitle(manga.title)
            .setCoverUrl(manga.thumbnail_url ?: "")
            .build()

    private fun toEntrySummaryAnime(anime: SAnime): Sandbox.EntrySummary =
        Sandbox.EntrySummary.newBuilder()
            .setSourceEntryId(anime.url)
            .setTitle(anime.title)
            .setCoverUrl(anime.thumbnail_url ?: "")
            .build()

    private fun mangaStatusToString(status: Int): String =
        when (status) {
            SManga.ONGOING -> "Ongoing"
            SManga.COMPLETED -> "Completed"
            SManga.LICENSED -> "Licensed"
            SManga.PUBLISHING_FINISHED -> "Publishing Finished"
            SManga.CANCELLED -> "Cancelled"
            SManga.ON_HIATUS -> "On Hiatus"
            else -> ""
        }

    private fun toEntryDetails(manga: SManga): Sandbox.EntryDetails =
        Sandbox.EntryDetails.newBuilder()
            .setSourceEntryId(manga.url)
            .setTitle(manga.title)
            .setDescription(manga.description ?: "")
            .setCoverUrl(manga.thumbnail_url ?: "")
            .addAllAuthors(listOfNotNull(manga.author).filter { it.isNotBlank() })
            .addAllGenres(manga.genre?.split(",")?.map { it.trim() } ?: emptyList())
            .setStatus(mangaStatusToString(manga.status))
            .addAllArtists(listOfNotNull(manga.artist).filter { it.isNotBlank() })
            .build()

    private fun toEntryDetailsNovel(novel: SourceNovel): Sandbox.EntryDetails =
        Sandbox.EntryDetails.newBuilder()
            .setSourceEntryId(novel.path)
            .setTitle(novel.name)
            .setDescription(novel.summary ?: "")
            .setCoverUrl(novel.cover ?: "")
            .addAllAuthors(listOfNotNull(novel.author))
            .addAllGenres(novel.genres?.split(",")?.map { it.trim() } ?: emptyList())
            .setStatus(novel.status ?: "")
            .build()
    private fun animeStatusToString(status: Int): String =
        when (status) {
            SAnime.ONGOING -> "Ongoing"
            SAnime.COMPLETED -> "Completed"
            SAnime.LICENSED -> "Licensed"
            SAnime.PUBLISHING_FINISHED -> "Publishing Finished"
            SAnime.CANCELLED -> "Cancelled"
            SAnime.ON_HIATUS -> "On Hiatus"
            else -> ""
        }

    private fun SAnime.mergeDetailsFrom(from: SAnime) {
        fun <T> orNull(get: () -> T): T? =
            try { get() } catch (_: UninitializedPropertyAccessException) { null }

        orNull { from.url }?.takeIf { it.isNotBlank() }?.let { url = it }
        orNull { from.title }?.takeIf { it.isNotBlank() }?.let { title = it }
        from.artist?.let { artist = it }
        from.author?.let { author = it }
        from.description?.let { description = it }
        from.genre?.let { genre = it }
        if (from.status != 0) status = from.status
        from.thumbnail_url?.let { thumbnail_url = it }
        initialized = true
    }

    private fun toEntryDetailsAnime(anime: SAnime): Sandbox.EntryDetails =
        Sandbox.EntryDetails.newBuilder()
            .setSourceEntryId(anime.url)
            .setTitle(anime.title)
            .setDescription(anime.description ?: "")
            .setCoverUrl(anime.thumbnail_url ?: "")
            .addAllAuthors(listOfNotNull(anime.author))
            .addAllGenres(anime.genre?.split(",")?.map { it.trim() } ?: emptyList())
            .setStatus(animeStatusToString(anime.status))
            .build()

    private fun toChapterSummary(chapter: SChapter): Sandbox.ChapterSummary =
        Sandbox.ChapterSummary.newBuilder()
            .setSourceChapterId(chapter.url)
            .setName(chapter.name)
            .setNumber(chapter.chapter_number.toDouble())
            .setUploadTimestamp(chapter.date_upload)
            .build()

    private fun toChapterSummaryNovel(chapter: ChapterItem): Sandbox.ChapterSummary =
        Sandbox.ChapterSummary.newBuilder()
            .setSourceChapterId(chapter.path)
            .setName(chapter.name)
            .setNumber(chapter.chapterNumber ?: 0.0)
            .setUploadTimestamp(0)
            .build()

    private fun toEpisodeSummary(episode: SEpisode): Sandbox.EpisodeSummary =
        Sandbox.EpisodeSummary.newBuilder()
            .setSourceEpisodeId(episode.url)
            .setName(episode.name)
            .setNumber(episode.episode_number.toDouble())
            .setUploadTimestamp(episode.date_upload)
            .build()

    private val resolutionRe = Regex("""(\d{3,4})\s*[pP]""")

    private fun parseResolution(video: Video): Int =
        video.resolution ?: resolutionRe.find(video.videoTitle)?.groupValues?.get(1)?.toIntOrNull() ?: 0

    private fun subtitleProtos(video: Video): List<Sandbox.SubtitleTrack> =
        video.subtitleTracks.map {
            Sandbox.SubtitleTrack.newBuilder().setUrl(it.url).setLang(it.lang).build()
        }

    private fun headerMap(video: Video): Map<String, String> =
        video.headers?.let { h -> h.names().associateWith { h[it] ?: "" } } ?: emptyMap()

    private fun toStreamInfo(preferred: Video, all: List<Video>): Sandbox.StreamInfo {
        val builder = Sandbox.StreamInfo.newBuilder()
            .setStreamUrl(preferred.videoUrl)
            .setQuality(preferred.videoTitle)

        builder.addAllSubtitles(subtitleProtos(preferred))
        headerMap(preferred).forEach { (k, v) -> builder.putHeaders(k, v) }

        all.forEach { v ->
            val src = Sandbox.VideoSource.newBuilder()
                .setUrl(v.videoUrl)
                .setLabel(v.videoTitle)
                .setResolution(parseResolution(v))
                .setPreferred(v === preferred || v.preferred)
                .addAllSubtitles(subtitleProtos(v))
            headerMap(v).forEach { (k, value) -> src.putHeaders(k, value) }
            builder.addSources(src.build())
        }

        preferred.audioTracks.forEach { t ->
            builder.addAudioTracks(
                Sandbox.AudioTrack.newBuilder().setUrl(t.url).setLang(t.lang).build(),
            )
        }

        preferred.timestamps.forEach { ts ->
            builder.addTimestamps(
                Sandbox.SkipTimestamp.newBuilder()
                    .setStartMs(ts.start)
                    .setEndMs(ts.end)
                    .setName(ts.name)
                    .setType(ts.type.name)
                    .build(),
            )
        }

        return builder.build()
    }

    override fun peekExtension(
        request: Sandbox.PeekExtensionRequest,
        responseObserver: StreamObserver<Sandbox.ExtensionMetadata>,
    ) {
        try {
            val sourceFile = File(request.filePath)
            if (!sourceFile.exists()) {
                responseObserver.onError(
                    StatusRuntimeException(Status.NOT_FOUND.withDescription("file not found: ${request.filePath}")),
                )
                return
            }

            val ext = sourceFile.extension.ifBlank { "apk" }
            val packageName = if (ext == "js") {
                sourceFile.nameWithoutExtension
            } else {
                ExtensionLoader.peekPackageName(sourceFile)
            }

            val targetFile = registry.install(sourceFile, packageName)
            val loaded = targetFile

            val lang = extensionLang(loaded.source)
            val name = extensionName(loaded.source) ?: loaded.packageName

            val metadata = Sandbox.ExtensionMetadata.newBuilder()
                .setPackageName(loaded.packageName)
                .setName(name)
                .setContentType(toContentTypeProto(loaded.contentType))
                .setLang(lang)
                .build()

            responseObserver.onNext(metadata)
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "peek extension failed" }
            responseObserver.onError(internal(e))
        }
    }

    private fun extensionLang(source: Any): String =
        when (source) {
            is eu.kanade.tachiyomi.source.CatalogueSource -> source.lang
            is eu.kanade.tachiyomi.animesource.AnimeCatalogueSource -> source.lang
            else -> "all"
        }

    private fun extensionName(source: Any): String? =
        when (source) {
            is eu.kanade.tachiyomi.source.Source -> source.name
            is eu.kanade.tachiyomi.animesource.AnimeSource -> source.name
            else -> null
        }

    private fun notFound(extensionId: String) =
        StatusRuntimeException(Status.NOT_FOUND.withDescription("extension not found: $extensionId"))

    private fun internal(e: Throwable) =
        StatusRuntimeException(Status.INTERNAL.withDescription(e.toString()).withCause(e))

    private fun unimplemented() = StatusRuntimeException(Status.UNIMPLEMENTED)
}
