package androidx.preference

import android.content.Context

open class ListPreference @JvmOverloads constructor(context: Context? = null) : DialogPreference(context) {
    var entries: Array<out CharSequence>? = null
    var entryValues: Array<out CharSequence>? = null
    var value: String? = null

    fun findIndexOfValue(value: String?): Int =
        entryValues?.indexOfFirst { it.toString() == value } ?: -1
}
