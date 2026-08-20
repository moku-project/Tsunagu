package tsunagu.grpc

import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga
import eu.kanade.tachiyomi.source.online.HttpSource
import io.github.oshai.kotlinlogging.KotlinLogging
import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.grpc.stub.StreamObserver
import kotlinx.coroutines.runBlocking
import sandbox.v1.ExtensionServiceGrpc
import sandbox.v1.Sandbox
import tsunagu.loader.ContentType
import tsunagu.loader.LoadedExtension
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
        handle(responseObserver, request.extensionId) { source ->
            val page = runBlocking { source.getSearchManga(request.page, request.query, source.getFilterList()) }
            val builder = Sandbox.SearchResponse.newBuilder().setHasNextPage(page.hasNextPage)
            page.mangas.forEach { manga -> builder.addResults(toEntrySummary(manga)) }
            builder.build()
        }
    }

    override fun getDetails(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.EntryDetails>,
    ) {
        handle(responseObserver, request.extensionId) { source ->
            val stub = SManga.create().apply { url = request.sourceEntryId; title = "" }
            val update = runBlocking {
                source.getMangaUpdate(stub, emptyList(), fetchDetails = true, fetchChapters = false)
            }
            toEntryDetails(update.manga)
        }
    }

    override fun getChapters(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.ChapterList>,
    ) {
        handle(responseObserver, request.extensionId) { source ->
            val stub = SManga.create().apply { url = request.sourceEntryId; title = "" }
            val update = runBlocking {
                source.getMangaUpdate(stub, emptyList(), fetchDetails = false, fetchChapters = true)
            }
            val builder = Sandbox.ChapterList.newBuilder()
            update.chapters.forEach { chapter -> builder.addChapters(toChapterSummary(chapter)) }
            builder.build()
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
        responseObserver.onError(unimplemented())
    }

    override fun getVideoStream(
        request: Sandbox.EpisodeRequest,
        responseObserver: StreamObserver<Sandbox.StreamInfo>,
    ) {
        responseObserver.onError(unimplemented())
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

    private fun toEntryDetails(manga: SManga): Sandbox.EntryDetails =
        Sandbox.EntryDetails.newBuilder()
            .setSourceEntryId(manga.url)
            .setTitle(manga.title)
            .setDescription(manga.description ?: "")
            .setCoverUrl(manga.thumbnail_url ?: "")
            .addAllAuthors(listOfNotNull(manga.author))
            .addAllGenres(manga.genre?.split(",")?.map { it.trim() } ?: emptyList())
            .build()

    private fun toChapterSummary(chapter: SChapter): Sandbox.ChapterSummary =
        Sandbox.ChapterSummary.newBuilder()
            .setSourceChapterId(chapter.url)
            .setName(chapter.name)
            .setNumber(chapter.chapter_number.toDouble())
            .setUploadTimestamp(chapter.date_upload)
            .build()

    private fun notFound(extensionId: String) =
        StatusRuntimeException(Status.NOT_FOUND.withDescription("extension not found: $extensionId"))

    private fun internal(e: Throwable) =
        StatusRuntimeException(Status.INTERNAL.withDescription(e.toString()).withCause(e))

    private fun unimplemented() = StatusRuntimeException(Status.UNIMPLEMENTED)
}