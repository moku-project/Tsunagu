package android.webkit

fun interface ValueCallback<T> {
    fun onReceiveValue(value: T)
}
