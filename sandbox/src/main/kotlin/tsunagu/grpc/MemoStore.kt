package tsunagu.grpc

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import java.io.File
import java.util.concurrent.ConcurrentHashMap

// Extensions keep private state in SManga/SChapter/SAnime/SEpisode.memo (series
// ids, slugs) and need it back on later calls; Tsunagu only stores urls, so the
// memos are kept here, one append-only JSON-lines file per extension.
object MemoStore {
    @Serializable
    private data class Line(val k: String, val u: String, val m: JsonObject)

    private val json = Json { ignoreUnknownKeys = true }
    private val stores = ConcurrentHashMap<String, Store>()

    @Volatile
    var baseDir: File? = null

    fun get(extensionId: String, kind: String, url: String): JsonObject? =
        store(extensionId)?.get("$kind $url")

    fun put(extensionId: String, kind: String, url: String, memo: JsonObject?) {
        if (memo.isNullOrEmpty() || url.isBlank()) return
        store(extensionId)?.put(kind, url, memo)
    }

    private fun store(extensionId: String): Store? {
        val dir = baseDir ?: return null
        val file = File(dir, "$extensionId.jsonl")
        return stores.computeIfAbsent(file.path) { Store(file) }
    }

    private class Store(private val file: File) {
        private val memos: MutableMap<String, JsonObject> by lazy { load() }

        @Synchronized
        fun get(key: String): JsonObject? = memos[key]

        @Synchronized
        fun put(kind: String, url: String, memo: JsonObject) {
            val key = "$kind $url"
            if (memos[key] == memo) return
            memos[key] = memo
            file.parentFile?.mkdirs()
            file.appendText(json.encodeToString(Line.serializer(), Line(kind, url, memo)) + "\n")
        }

        private fun load(): MutableMap<String, JsonObject> {
            val map = HashMap<String, JsonObject>()
            if (!file.exists()) return map
            var lines = 0
            file.forEachLine { raw ->
                lines++
                runCatching { json.decodeFromString(Line.serializer(), raw) }
                    .onSuccess { map["${it.k} ${it.u}"] = it.m }
            }
            // Updates append, so rewrite once superseded lines dominate.
            if (lines > 2 * map.size + 100) {
                val tmp = File(file.path + ".tmp")
                tmp.bufferedWriter().use { w ->
                    map.forEach { (key, memo) ->
                        val (k, u) = key.split(' ', limit = 2)
                        w.write(json.encodeToString(Line.serializer(), Line(k, u, memo)))
                        w.newLine()
                    }
                }
                tmp.renameTo(file) || run { file.delete(); tmp.renameTo(file) }
            }
            return map
        }
    }
}
