package android.app

import android.content.Context
import android.content.ContextWrapper
import android.content.InMemorySharedPreferences
import android.content.SharedPreferences
import java.util.concurrent.ConcurrentHashMap

class Application : ContextWrapper(null) {
    private val prefs = ConcurrentHashMap<String, SharedPreferences>()
    override fun getSharedPreferences(name: String, mode: Int): SharedPreferences =
        prefs.getOrPut(name) { InMemorySharedPreferences() }
}
