package android.content

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray
import java.io.File

class FileBackedSharedPreferences(private val file: File) : SharedPreferences {

    private val json = Json { prettyPrint = true }
    private val lock = Any()
    private val store = LinkedHashMap<String, Any?>()

    init {
        runCatching {
            if (file.isFile) {
                val obj = json.parseToJsonElement(file.readText()).jsonObject
                for ((k, v) in obj) store[k] = decode(v as? JsonPrimitive ?: v)
            }
        }
    }

    private fun decode(v: Any?): Any? = when (v) {
        is JsonPrimitive -> when {
            v.isString -> v.content
            v.booleanOrNull != null -> v.booleanOrNull
            v.longOrNull != null -> v.longOrNull
            v.doubleOrNull != null -> v.doubleOrNull
            else -> v.contentOrNull
        }
        is JsonArray -> v.map { it.jsonPrimitive.content }.toSet()
        else -> null
    }

    private fun persist() {
        val obj: JsonObject = buildJsonObject {
            for ((k, v) in store) {
                when (v) {
                    is String -> put(k, v)
                    is Boolean -> put(k, v)
                    is Int -> put(k, v)
                    is Long -> put(k, v)
                    is Float -> put(k, v.toDouble())
                    is Double -> put(k, v)
                    is Set<*> -> putJsonArray(k) { v.forEach { add(JsonPrimitive(it.toString())) } }
                    else -> {}
                }
            }
        }
        file.parentFile?.mkdirs()
        val tmp = File(file.parentFile, file.name + ".tmp")
        tmp.writeText(json.encodeToString(JsonObject.serializer(), obj))
        tmp.renameTo(file)
    }

    override fun getString(key: String, defValue: String?) =
        synchronized(lock) { store[key] as? String ?: defValue }

    override fun getInt(key: String, defValue: Int) = synchronized(lock) {
        when (val v = store[key]) {
            is Int -> v
            is Long -> v.toInt()
            is Number -> v.toInt()
            else -> defValue
        }
    }

    override fun getLong(key: String, defValue: Long) = synchronized(lock) {
        when (val v = store[key]) {
            is Long -> v
            is Int -> v.toLong()
            is Number -> v.toLong()
            else -> defValue
        }
    }

    override fun getFloat(key: String, defValue: Float) = synchronized(lock) {
        when (val v = store[key]) {
            is Float -> v
            is Number -> v.toFloat()
            else -> defValue
        }
    }

    override fun getBoolean(key: String, defValue: Boolean) =
        synchronized(lock) { store[key] as? Boolean ?: defValue }

    @Suppress("UNCHECKED_CAST")
    override fun getStringSet(key: String, defValue: Set<String>?) =
        synchronized(lock) { store[key] as? Set<String> ?: defValue }

    override fun contains(key: String) = synchronized(lock) { store.containsKey(key) }

    override fun getAll(): Map<String, *> = synchronized(lock) { LinkedHashMap(store) }

    override fun edit(): SharedPreferences.Editor = EditorImpl()

    private inner class EditorImpl : SharedPreferences.Editor {
        private val pending = LinkedHashMap<String, Any?>()
        private var clearFlag = false

        override fun putString(key: String, value: String?) = apply { pending[key] = value }
        override fun putInt(key: String, value: Int) = apply { pending[key] = value }
        override fun putBoolean(key: String, value: Boolean) = apply { pending[key] = value }
        override fun putLong(key: String, value: Long) = apply { pending[key] = value }
        override fun putFloat(key: String, value: Float) = apply { pending[key] = value }
        override fun putStringSet(key: String, values: Set<String>?) = apply { pending[key] = values?.toSet() }
        override fun remove(key: String) = apply { pending[key] = REMOVE }
        override fun clear() = apply { clearFlag = true }

        override fun apply() { commit() }

        override fun commit(): Boolean {
            synchronized(lock) {
                if (clearFlag) store.clear()
                for ((k, v) in pending) if (v === REMOVE) store.remove(k) else store[k] = v
                runCatching { persist() }
            }
            return true
        }
    }

    private companion object {
        val REMOVE = Any()
    }
}
