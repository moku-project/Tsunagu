package tsunagu.grpc

import io.grpc.Status
import io.grpc.StatusRuntimeException
import io.grpc.stub.StreamObserver
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.runBlocking
import sandbox.v1.ExtensionServiceGrpc
import sandbox.v1.Sandbox
import tsunagu.loader.LoadedExtension
import tsunagu.registry.ExtensionRegistry
import tsunagu.source.FilterList
import tsunagu.source.SManga
import tsunagu.source.SChapter
import tsunagu.source.SourceBridge

class ExtensionServiceImpl(private val registry: ExtensionRegistry) : ExtensionServiceGrpc.ExtensionServiceImplBase() {
    private val logger = KotlinLogging.logger {}

    override fun addRepository(
        request: Sandbox.AddRepositoryRequest,
        responseObserver: StreamObserver<Sandbox.Repository>,
    ) {
        responseObserver.onError(unimplemented())
    }

    override fun listRepositories(
        request: Sandbox.Empty,
        responseObserver: StreamObserver<Sandbox.RepositoryList>,
    ) {
        responseObserver.onError(unimplemented())
    }

    override fun listAvailableExtensions(
        request: Sandbox.ListAvailableExtensionsRequest,
        responseObserver: StreamObserver<Sandbox.ExtensionList>,
    ) {
        responseObserver.onError(unimplemented())
    }

    override fun installExtension(
        request: Sandbox.InstallExtensionRequest,
        responseObserver: StreamObserver<Sandbox.Extension>,
    ) {
        responseObserver.onError(unimplemented())
    }

    override fun listInstalledExtensions(
        request: Sandbox.Empty,
        responseObserver: StreamObserver<Sandbox.ExtensionList>,
    ) {
        val builder = Sandbox.ExtensionList.newBuilder()
        registry.list().forEach { ext -> builder.addExtensions(toExtensionProto(ext)) }
        responseObserver.onNext(builder.build())
        responseObserver.onCompleted()
    }

    override fun uninstallExtension(
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
        handle(responseObserver, request.extensionId) { bridge ->
            val results = runBlocking { bridge.getSearchManga(request.page, request.query, FilterList()) }
            val builder = Sandbox.SearchResponse.newBuilder().setHasNextPage(false)
            results.forEach { manga -> builder.addResults(toEntrySummary(manga)) }
            builder.build()
        }
    }

    override fun getDetails(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.EntryDetails>,
    ) {
        handle(responseObserver, request.extensionId) { bridge ->
            val manga = runBlocking { bridge.getMangaDetails(SManga(url = request.sourceEntryId, title = "")) }
            toEntryDetails(manga)
        }
    }

    override fun getChapters(
        request: Sandbox.EntryRequest,
        responseObserver: StreamObserver<Sandbox.ChapterList>,
    ) {
        handle(responseObserver, request.extensionId) { bridge ->
            val chapters = runBlocking { bridge.getChapterList(SManga(url = request.sourceEntryId, title = "")) }
            val builder = Sandbox.ChapterList.newBuilder()
            chapters.forEach { chapter -> builder.addChapters(toChapterSummary(chapter)) }
            builder.build()
        }
    }

    override fun getPages(
        request: Sandbox.ChapterRequest,
        responseObserver: StreamObserver<Sandbox.PageList>,
    ) {
        handle(responseObserver, request.extensionId) { bridge ->
            val chapter = SChapter(url = request.sourceChapterId, name = "")
            val pages = runBlocking { bridge.getPageList(chapter) }
            val builder = Sandbox.PageList.newBuilder()
            pages.forEach { page -> builder.addPageUrls(page.imageUrl ?: page.url) }
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

    private fun <T> handle(responseObserver: StreamObserver<T>, extensionId: String, block: (SourceBridge) -> T) {
        val extension = registry.get(extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(extensionId))
            return
        }
        try {
            responseObserver.onNext(block(SourceBridge(extension)))
            responseObserver.onCompleted()
        } catch (e: Throwable) {
            logger.error(e) { "extension call failed" }
            responseObserver.onError(internal(e))
        }
    }

    private fun toExtensionProto(ext: LoadedExtension): Sandbox.Extension =
        Sandbox.Extension.newBuilder()
            .setId(ext.packageName)
            .setName(ext.packageName)
            .setContentType(Sandbox.ContentType.MANGA)
            .build()

    private fun toEntrySummary(manga: SManga): Sandbox.EntrySummary =
        Sandbox.EntrySummary.newBuilder()
            .setSourceEntryId(manga.url)
            .setTitle(manga.title)
            .setCoverUrl(manga.thumbnailUrl ?: "")
            .build()

    private fun toEntryDetails(manga: SManga): Sandbox.EntryDetails =
        Sandbox.EntryDetails.newBuilder()
            .setSourceEntryId(manga.url)
            .setTitle(manga.title)
            .setDescription(manga.description ?: "")
            .setCoverUrl(manga.thumbnailUrl ?: "")
            .addAllAuthors(listOfNotNull(manga.author))
            .addAllGenres(manga.genre?.split(",")?.map { it.trim() } ?: emptyList())
            .build()

    private fun toChapterSummary(chapter: SChapter): Sandbox.ChapterSummary =
        Sandbox.ChapterSummary.newBuilder()
            .setSourceChapterId(chapter.url)
            .setName(chapter.name)
            .setNumber(chapter.chapterNumber.toDouble())
            .setUploadTimestamp(chapter.dateUpload)
            .build()

    private fun notFound(extensionId: String) =
        StatusRuntimeException(Status.NOT_FOUND.withDescription("extension not found: $extensionId"))

    private fun internal(e: Throwable) =
        StatusRuntimeException(Status.INTERNAL.withDescription(e.toString()).withCause(e))

    private fun unimplemented() = StatusRuntimeException(Status.UNIMPLEMENTED)
}
