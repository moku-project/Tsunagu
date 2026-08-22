package tsunagu.novel

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonPrimitive
import java.io.File
import java.util.concurrent.ConcurrentHashMap

object PluginStorage {

    private val json = Json { ignoreUnknownKeys = true }
    private val locks = ConcurrentHashMap<String, Any>()

    var baseDir: File = File("plugin-storage")

    fun get(namespace: String, key: String): Any? {
        val store = readStore(namespace)
        return store[key]?.toKotlinValue()
    }

    fun set(namespace: String, key: String, value: Any?) {
        val lock = locks.computeIfAbsent(namespace) { Any() }
        synchronized(lock) {
            val store = readStore(namespace).toMutableMap()
            store[key] = value.toJsonElement()
            writeStore(namespace, store)
        }
    }

    fun delete(namespace: String, key: String) {
        val lock = locks.computeIfAbsent(namespace) { Any() }
        synchronized(lock) {
            val store = readStore(namespace).toMutableMap()
            store.remove(key)
            writeStore(namespace, store)
        }
    }

    private fun storeFile(namespace: String): File {
        val safeName = namespace.replace(Regex("[^a-zA-Z0-9._-]"), "_")
        return File(baseDir, "$safeName.json")
    }

    private fun readStore(namespace: String): Map<String, JsonElement> {
        val file = storeFile(namespace)
        if (!file.exists()) return emptyMap()
        return try {
            json.decodeFromString<Map<String, JsonElement>>(file.readText())
        } catch (e: Exception) {
            emptyMap()
        }
    }

    private fun writeStore(namespace: String, store: Map<String, JsonElement>) {
        val file = storeFile(namespace)
        file.parentFile?.mkdirs()
        file.writeText(json.encodeToString(store))
    }

    private fun JsonElement.toKotlinValue(): Any? = when (this) {
        is JsonNull -> null
        is JsonPrimitive -> when {
            booleanOrNull != null -> boolean
            doubleOrNull != null && !isString -> doubleOrNull
            else -> contentOrNull
        }
        else -> null
    }

    private fun Any?.toJsonElement(): JsonElement = when (this) {
        null -> JsonNull
        is Boolean -> JsonPrimitive(this)
        is Int -> JsonPrimitive(this)
        is Long -> JsonPrimitive(this)
        is Double -> JsonPrimitive(this)
        is String -> JsonPrimitive(this)
        else -> JsonPrimitive(this.toString())
    }
}