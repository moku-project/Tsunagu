package androidx.preference

import android.widget.EditText

open class EditTextPreference : Preference() {
    interface OnBindEditTextListener {
        fun onBindEditText(editText: EditText)
    }
}
