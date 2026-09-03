package androidx.preference

import android.content.Context

open class Preference @JvmOverloads constructor(val context: Context? = null) {

    var key: String? = null
    var title: CharSequence? = null
    var summary: CharSequence? = null
    var isVisible: Boolean = true
    var isEnabled: Boolean = true
    var isPersistent: Boolean = true
    var isIconSpaceReserved: Boolean = false
    var order: Int = 0

    @JvmField
    var defaultValue: Any? = null

    var onPreferenceChangeListener: OnPreferenceChangeListener? = null
    var onPreferenceClickListener: OnPreferenceClickListener? = null

    open fun setDefaultValue(value: Any?) {
        defaultValue = value
    }

    fun interface OnPreferenceChangeListener {
        fun onPreferenceChange(preference: Preference, newValue: Any?): Boolean
    }

    fun interface OnPreferenceClickListener {
        fun onPreferenceClick(preference: Preference): Boolean
    }
}
