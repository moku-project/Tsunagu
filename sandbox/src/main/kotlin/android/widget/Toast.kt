package android.widget

class Toast {
    companion object {
        const val LENGTH_SHORT = 0
        const val LENGTH_LONG = 1
        fun makeText(context: Any?, text: CharSequence?, duration: Int): Toast = Toast()
    }
    fun show() {}
}
