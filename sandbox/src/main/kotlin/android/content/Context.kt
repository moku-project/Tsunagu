package android.content

import java.io.File

abstract class Context {
    abstract fun getSharedPreferences(name: String, mode: Int): SharedPreferences

    open fun getApplicationContext(): Context = this
    open fun getPackageName(): String = "tsunagu.sandbox"

    open fun getCacheDir(): File = sandboxDir("cache")
    open fun getCodeCacheDir(): File = sandboxDir("code_cache")
    open fun getExternalCacheDir(): File? = sandboxDir("cache")
    open fun getFilesDir(): File = sandboxDir("files")
    open fun getNoBackupFilesDir(): File = sandboxDir("no_backup")
    open fun getExternalFilesDir(type: String?): File? =
        sandboxDir(if (type.isNullOrEmpty()) "files" else "files/$type")
    open fun getDir(name: String, mode: Int): File = sandboxDir("app_$name")

    open fun getSystemService(name: String): Any? = null
    open fun getString(resId: Int): String = ""
    open fun getString(resId: Int, vararg formatArgs: Any?): String = ""

    companion object {
        const val MODE_PRIVATE = 0
        const val MODE_APPEND = 0x8000
        const val MODE_MULTI_PROCESS = 0x0004
        const val MODE_WORLD_READABLE = 0x0001
        const val MODE_WORLD_WRITEABLE = 0x0002

        private val root: File by lazy {
            File(System.getProperty("java.io.tmpdir") ?: ".", "tsunagu-android-ctx").apply { mkdirs() }
        }

        private fun sandboxDir(sub: String): File = File(root, sub).apply { mkdirs() }
    }
}
