package androidx.preference

import android.content.Context

open class SeekBarPreference @JvmOverloads constructor(context: Context? = null) : Preference(context) {
    var value: Int = 0
    var min: Int = 0
    var max: Int = 100
}
