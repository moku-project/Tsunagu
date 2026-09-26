package tsunagu.grpc

import eu.kanade.tachiyomi.source.model.Page
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit

// Some sources return page links that each require another HTTP request.
// Bound that work without serializing the whole chapter or changing page order.
internal suspend fun resolvePageURLs(
    pages: List<Page>,
    resolve: suspend (Page) -> String,
): List<String> = coroutineScope {
    val slots = Semaphore(3)
    pages.map { page ->
        async(Dispatchers.IO) {
            page.imageUrl ?: slots.withPermit { resolve(page) }
        }
    }.awaitAll()
}
