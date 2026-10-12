package tsunagu.grpc

import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import java.io.File
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class MemoStoreTest {
    private fun memo(id: Int) = JsonObject(mapOf("id" to JsonPrimitive(id)))

    private fun freshDir(): File = Files.createTempDirectory("memo").toFile().also { it.deleteOnExit() }

    @Test
    fun survivesRestart() {
        val first = freshDir()
        MemoStore.baseDir = first
        MemoStore.put("ext", "chapter", "/c/1", memo(1))
        MemoStore.put("ext", "chapter", "/c/1", memo(2))
        MemoStore.put("ext", "manga", "/c/1", memo(3))
        MemoStore.put("ext", "chapter", "/c/2", JsonObject(emptyMap())) // empty memos aren't stored

        // A restarted sandbox reads the same file with nothing in memory.
        val restarted = freshDir()
        File(first, "ext.jsonl").copyTo(File(restarted, "ext.jsonl"))
        MemoStore.baseDir = restarted
        assertEquals(memo(2), MemoStore.get("ext", "chapter", "/c/1"))
        assertEquals(memo(3), MemoStore.get("ext", "manga", "/c/1"))
        assertNull(MemoStore.get("ext", "chapter", "/c/2"))
        assertNull(MemoStore.get("other", "chapter", "/c/1"))
    }

    @Test
    fun compactsSupersededLines() {
        val dir = freshDir()
        MemoStore.baseDir = dir
        repeat(300) { MemoStore.put("ext", "chapter", "/c/1", memo(it)) }

        val restarted = freshDir()
        val file = File(restarted, "ext.jsonl")
        File(dir, "ext.jsonl").copyTo(file)
        MemoStore.baseDir = restarted
        assertEquals(memo(299), MemoStore.get("ext", "chapter", "/c/1"))
        assertEquals(1, file.readLines().size)
    }
}
