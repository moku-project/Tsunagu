package android.app

import android.content.ContextWrapper
import android.content.FileBackedSharedPreferences
import android.content.SharedPreferences
import java.io.File
import java.util.concurrent.ConcurrentHashMap

class Application : ContextWrapper(null) {
    private val prefs = ConcurrentHashMap<String, SharedPreferences>()

    override fun getSharedPreferences(name: String, mode: Int): SharedPreferences =
        prefs.getOrPut(name) {
            val safe = name.replace(Regex("[^A-Za-z0-9._-]"), "_")
            FileBackedSharedPreferences(File(prefsRoot, "$safe.json"))
        }

    companion object {
        @JvmStatic
        var prefsRoot: File = File("shared-prefs")
    }
}
