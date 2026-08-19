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
import tsunagu.loader.ContentTypeClassifier
import tsunagu.loader.LoadedExtension
import tsunagu.registry.ExtensionRegistry
import tsunagu.repository.ParsedExtension
import tsunagu.repository.RepositoryRegistry
import tsunagu.source.getMangaUpdate
import tsunagu.source.getPageList
import tsunagu.source.getSearchManga

class ExtensionServiceImpl(
    private val registry: ExtensionRegistry,
    private val repositoryRegistry: RepositoryRegistry,
) : ExtensionServiceGrpc.ExtensionServiceImplBase() {

    private val logger = KotlinLogging.logger {}

    override fun addRepository(
        request: Sandbox.AddRepositoryRequest,
        responseObserver: StreamObserver<Sandbox.Repository>,
    ) {
        try {
            val entry = repositoryRegistry.add(request.indexUrl)
            responseObserver.onNext(
                Sandbox.Repository.newBuilder()
                    .setId(entry.id)
                    .setIndexUrl(entry.indexUrl)
                    .setName(entry.indexUrl)
                    .build()
            )
            responseObserver.onCompleted()
        } catch (e: Exception) {
            logger.error(e) { "add repository failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun listRepositories(
        request: Sandbox.Empty,
        responseObserver: StreamObserver<Sandbox.RepositoryList>,
    ) {
        val builder = Sandbox.RepositoryList.newBuilder()
        repositoryRegistry.list().forEach { entry ->
            builder.addRepositories(
                Sandbox.Repository.newBuilder()
                    .setId(entry.id)
                    .setIndexUrl(entry.indexUrl)
                    .setName(entry.indexUrl)
                    .build()
            )
        }
        responseObserver.onNext(builder.build())
        responseObserver.onCompleted()
    }

    override fun listAvailableExtensions(
        request: Sandbox.ListAvailableExtensionsRequest,
        responseObserver: StreamObserver<Sandbox.ExtensionList>,
    ) {
        try {
            val repo = repositoryRegistry.get(request.repositoryId)
            val builder = Sandbox.ExtensionList.newBuilder()
            repo.extensions.forEach { ext -> builder.addExtensions(toAvailableExtensionProto(ext)) }
            responseObserver.onNext(builder.build())
            responseObserver.onCompleted()
        } catch (e: Exception) {
            logger.error(e) { "list available extensions failed" }
            responseObserver.onError(internal(e))
        }
    }

    override fun installExtension(
        request: Sandbox.InstallExtensionRequest,
        responseObserver: StreamObserver<Sandbox.Extension>,
    ) {
        try {
            val match = repositoryRegistry.findExtension(request.repositoryId, request.extensionId)
            val loaded = registry.installFromUrl(match.apkUrl, match.jarUrl, match.packageName)
            responseObserver.onNext(toExtensionProto(loaded))
            responseObserver.onCompleted()
        } catch (e: Exception) {
            logger.error(e) { "install extension failed" }
            responseObserver.onError(internal(e))
        }
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
                    val imageUrl = page.imageUrl ?: (source as? HttpSource)?.getImageUrl(page)
                    imageUrl ?: page.url
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
        block: (eu.kanade.tachiyomi.source.online.HttpSource) -> T,
    ) {
        val extension = registry.get(extensionId)
        if (extension == null) {
            responseObserver.onError(notFound(extensionId))
            return
        }
        try {
            val httpSource = extension.source as? eu.kanade.tachiyomi.source.online.HttpSource
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

    private fun toAvailableExtensionProto(ext: ParsedExtension): Sandbox.Extension =
        Sandbox.Extension.newBuilder()
            .setId(ext.packageName)
            .setName(ext.name)
            .setVersion(ext.versionName)
            .setContentType(toContentTypeProto(ContentTypeClassifier.fromPackageName(ext.packageName)))
            .setLang(ext.lang)
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
