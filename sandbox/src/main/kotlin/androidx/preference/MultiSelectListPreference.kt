package androidx.preference

import android.content.Context

open class MultiSelectListPreference @JvmOverloads constructor(context: Context? = null) : DialogPreference(context) {
    var entries: Array<out CharSequence>? = null
    var entryValues: Array<out CharSequence>? = null
    var values: Set<String> = emptySet()
}
