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
import tsunagu.loader.LoadedExtension
import tsunagu.novel.ChapterItem
import tsunagu.novel.NovelPlugin
import tsunagu.novel.SourceNovel
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
                    val page = runBlocking { source.getSearchManga(request.page, request.query, source.getFilterList()) }
                    builder.setHasNextPage(page.hasNextPage)
                    page.mangas.forEach { manga -> builder.addResults(toEntrySummary(manga)) }
                }
                is AnimeHttpSource -> {
                    val page = runBlocking { source.getSearchAnime(request.page, request.query, source.getFilterList()) }
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
                    val anime = runBlocking { source.getAnimeDetails(stub) }
                    toEntryDetailsAnime(anime)
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
            val video = videos.firstOrNull { it.preferred }
                ?: videos.firstOrNull { it.videoUrl.contains("localhost") || it.videoUrl.contains("127.0.0.1") }
                ?: videos.first()
            toStreamInfo(video)
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

    private fun toExtensionProto(ext: LoadedExtension): Sandbox.Extension =
        Sandbox.Extension.newBuilder()
            .setId(ext.packageName)
            .setName(ext.packageName)
            .setContentType(toContentTypeProto(ext.contentType))
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

    private fun toEntryDetails(manga: SManga): Sandbox.EntryDetails =
        Sandbox.EntryDetails.newBuilder()
            .setSourceEntryId(manga.url)
            .setTitle(manga.title)
            .setDescription(manga.description ?: "")
            .setCoverUrl(manga.thumbnail_url ?: "")
            .addAllAuthors(listOfNotNull(manga.author))
            .addAllGenres(manga.genre?.split(",")?.map { it.trim() } ?: emptyList())
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

    private fun toStreamInfo(video: Video): Sandbox.StreamInfo {
        val builder = Sandbox.StreamInfo.newBuilder()
            .setStreamUrl(video.videoUrl)
            .setQuality(video.videoTitle)
        video.subtitleTracks.forEach { track ->
            builder.addSubtitles(
                Sandbox.SubtitleTrack.newBuilder()
                    .setUrl(track.url)
                    .setLang(track.lang)
                    .build(),
            )
        }
        video.headers?.let { headers ->
            for (name in headers.names()) {
                builder.putHeaders(name, headers[name] ?: "")
            }
        }
        return builder.build()
    }

    private fun notFound(extensionId: String) =
        StatusRuntimeException(Status.NOT_FOUND.withDescription("extension not found: $extensionId"))

    private fun internal(e: Throwable) =
        StatusRuntimeException(Status.INTERNAL.withDescription(e.toString()).withCause(e))

    private fun unimplemented() = StatusRuntimeException(Status.UNIMPLEMENTED)
}
