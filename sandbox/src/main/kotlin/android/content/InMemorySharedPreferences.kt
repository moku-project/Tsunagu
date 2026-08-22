package android.content

import java.util.concurrent.ConcurrentHashMap

class InMemorySharedPreferences : SharedPreferences {
    private val store = ConcurrentHashMap<String, Any?>()

    override fun getString(key: String, defValue: String?) = store[key] as? String ?: defValue
    override fun getInt(key: String, defValue: Int) = store[key] as? Int ?: defValue
    override fun getBoolean(key: String, defValue: Boolean) = store[key] as? Boolean ?: defValue
    override fun getLong(key: String, defValue: Long) = store[key] as? Long ?: defValue
    override fun getFloat(key: String, defValue: Float) = store[key] as? Float ?: defValue
    @Suppress("UNCHECKED_CAST")
    override fun getStringSet(key: String, defValue: Set<String>?) = store[key] as? Set<String> ?: defValue
    override fun contains(key: String) = store.containsKey(key)
    override fun getAll(): Map<String, *> = store.toMap()

    override fun edit(): SharedPreferences.Editor = EditorImpl(store)

    private class EditorImpl(
        private val store: ConcurrentHashMap<String, Any?>,
    ) : SharedPreferences.Editor {
        private val pending = mutableMapOf<String, Any?>()
        private var clearFlag = false

        override fun putString(key: String, value: String?) = apply { pending[key] = value }
        override fun putInt(key: String, value: Int) = apply { pending[key] = value }
        override fun putBoolean(key: String, value: Boolean) = apply { pending[key] = value }
        override fun putLong(key: String, value: Long) = apply { pending[key] = value }
        override fun putFloat(key: String, value: Float) = apply { pending[key] = value }
        override fun putStringSet(key: String, values: Set<String>?) = apply { pending[key] = values }
        override fun remove(key: String) = apply { pending[key] = REMOVE_MARKER }
        override fun clear() = apply { clearFlag = true }

        override fun apply() { commit() }

        override fun commit(): Boolean {
            if (clearFlag) store.clear()
            pending.forEach { (k, v) -> if (v === REMOVE_MARKER) store.remove(k) else store[k] = v }
            return true
        }

        companion object { private val REMOVE_MARKER = Any() }
    }
}
