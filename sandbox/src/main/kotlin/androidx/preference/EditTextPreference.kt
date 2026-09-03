package androidx.preference

import android.content.Context
import android.widget.EditText

open class EditTextPreference @JvmOverloads constructor(context: Context? = null) : DialogPreference(context) {
    var text: String? = null
    var onBindEditTextListener: OnBindEditTextListener? = null

    fun interface OnBindEditTextListener {
        fun onBindEditText(editText: EditText)
    }
}
