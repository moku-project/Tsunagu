package androidx.preference

import android.content.Context

open class DialogPreference @JvmOverloads constructor(context: Context? = null) : Preference(context) {
    var dialogTitle: CharSequence? = null
    var dialogMessage: CharSequence? = null
}
