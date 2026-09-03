package androidx.preference

import android.content.Context

open class CheckBoxPreference @JvmOverloads constructor(context: Context? = null) : Preference(context) {
    var isChecked: Boolean = false
    var summaryOn: CharSequence? = null
    var summaryOff: CharSequence? = null
}
