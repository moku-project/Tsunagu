package android.content

open class ContextWrapper(private val base: Context?) : Context() {
    override fun getSharedPreferences(name: String, mode: Int): SharedPreferences =
        base?.getSharedPreferences(name, mode)
            ?: throw IllegalStateException("ContextWrapper has no base Context")
}
