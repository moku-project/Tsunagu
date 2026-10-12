package tsunagu.source

import eu.kanade.tachiyomi.source.model.Page
import eu.kanade.tachiyomi.source.online.HttpSource
import java.lang.reflect.InvocationTargetException
import java.lang.reflect.Method
import java.util.concurrent.ConcurrentHashMap
import kotlin.coroutines.Continuation
import kotlin.coroutines.intrinsics.suspendCoroutineUninterceptedOrReturn

/**
 * Tsundoku-format novel extensions are manga HttpSources that also implement
 * Tsundoku's NovelSource, whose `suspend fun fetchPageText(page: Page): String`
 * returns a page's text as HTML. ProGuard strips the NovelSource interface from
 * the extension jars, so sources are recognized by that method instead.
 */
object NovelSources {
    private val methods = ConcurrentHashMap<Class<*>, Any>()
    private val none = Any()

    private fun fetchPageText(cls: Class<*>): Method? =
        methods.getOrPut(cls) {
            runCatching {
                cls.getMethod("fetchPageText", Page::class.java, Continuation::class.java)
                    .takeIf { it.returnType == Any::class.java }
            }.getOrNull() ?: none
        } as? Method

    fun isNovel(cls: Class<*>): Boolean = fetchPageText(cls) != null

    /** The chapter's text as HTML: every page of [pages] fetched in order. */
    suspend fun chapterText(source: HttpSource, pages: List<Page>): String {
        val method = fetchPageText(source.javaClass)
            ?: throw IllegalStateException("${source.name} is not a novel source")
        return pages.map { page -> method.invokeSuspend(source, page) as String }
            .joinToString("\n")
    }

    private suspend fun Method.invokeSuspend(target: Any, vararg args: Any?): Any? =
        suspendCoroutineUninterceptedOrReturn { cont ->
            try {
                invoke(target, *args, cont)
            } catch (e: InvocationTargetException) {
                throw e.targetException
            }
        }
}
