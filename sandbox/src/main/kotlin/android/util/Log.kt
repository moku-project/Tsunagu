package android.util

object Log {
    const val VERBOSE = 2
    const val DEBUG = 3
    const val INFO = 4
    const val WARN = 5
    const val ERROR = 6
    const val ASSERT = 7

    private fun brief(tr: Throwable?): String =
        if (tr == null) "" else " (${tr.javaClass.simpleName}: ${tr.message?.take(200) ?: ""})"

    @JvmStatic fun v(tag: String?, msg: String): Int = out("V/$tag: $msg")
    @JvmStatic fun v(tag: String?, msg: String, tr: Throwable): Int = out("V/$tag: $msg${brief(tr)}")

    @JvmStatic fun d(tag: String?, msg: String): Int = out("D/$tag: $msg")
    @JvmStatic fun d(tag: String?, msg: String, tr: Throwable): Int = out("D/$tag: $msg${brief(tr)}")

    @JvmStatic fun i(tag: String?, msg: String): Int = out("I/$tag: $msg")
    @JvmStatic fun i(tag: String?, msg: String, tr: Throwable): Int = out("I/$tag: $msg${brief(tr)}")

    @JvmStatic fun w(tag: String?, msg: String): Int = out("W/$tag: $msg")
    @JvmStatic fun w(tag: String?, msg: String, tr: Throwable): Int = out("W/$tag: $msg${brief(tr)}")
    @JvmStatic fun w(tag: String?, tr: Throwable): Int = out("W/$tag:${brief(tr)}")

    @JvmStatic fun e(tag: String?, msg: String): Int = out("E/$tag: $msg")
    @JvmStatic fun e(tag: String?, msg: String, tr: Throwable): Int = out("E/$tag: $msg${brief(tr)}")

    @JvmStatic fun wtf(tag: String?, msg: String): Int = out("WTF/$tag: $msg")

    @JvmStatic fun getStackTraceString(tr: Throwable?): String = tr?.stackTraceToString() ?: ""

    @JvmStatic fun isLoggable(tag: String?, level: Int): Boolean = false

    private fun out(msg: String): Int {
        println(msg)
        return msg.length
    }
}
