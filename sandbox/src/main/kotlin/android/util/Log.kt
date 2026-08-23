package android.util

object Log {
    const val VERBOSE = 2
    const val DEBUG = 3
    const val INFO = 4
    const val WARN = 5
    const val ERROR = 6
    const val ASSERT = 7

    @JvmStatic
    fun v(tag: String?, msg: String): Int = println("V/$tag: $msg")

    @JvmStatic
    fun v(tag: String?, msg: String, tr: Throwable): Int = println("V/$tag: $msg\n${tr.stackTraceToString()}")

    @JvmStatic
    fun d(tag: String?, msg: String): Int = println("D/$tag: $msg")

    @JvmStatic
    fun d(tag: String?, msg: String, tr: Throwable): Int = println("D/$tag: $msg\n${tr.stackTraceToString()}")

    @JvmStatic
    fun i(tag: String?, msg: String): Int = println("I/$tag: $msg")

    @JvmStatic
    fun i(tag: String?, msg: String, tr: Throwable): Int = println("I/$tag: $msg\n${tr.stackTraceToString()}")

    @JvmStatic
    fun w(tag: String?, msg: String): Int = println("W/$tag: $msg")

    @JvmStatic
    fun w(tag: String?, msg: String, tr: Throwable): Int = println("W/$tag: $msg\n${tr.stackTraceToString()}")

    @JvmStatic
    fun w(tag: String?, tr: Throwable): Int = println("W/$tag: ${tr.stackTraceToString()}")

    @JvmStatic
    fun e(tag: String?, msg: String): Int = println("E/$tag: $msg")

    @JvmStatic
    fun e(tag: String?, msg: String, tr: Throwable): Int = println("E/$tag: $msg\n${tr.stackTraceToString()}")

    @JvmStatic
    fun wtf(tag: String?, msg: String): Int = println("WTF/$tag: $msg")

    @JvmStatic
    fun getStackTraceString(tr: Throwable?): String = tr?.stackTraceToString() ?: ""

    @JvmStatic
    fun isLoggable(tag: String?, level: Int): Boolean = true

    private fun println(msg: String): Int {
        kotlin.io.println(msg)
        return msg.length
    }
}
