package androidx.preference

import android.content.Context

open class PreferenceGroup @JvmOverloads constructor(context: Context? = null) : Preference(context) {
    @JvmField
    val preferences = mutableListOf<Preference>()

    fun addPreference(preference: Preference): Boolean {
        preferences.add(preference)
        return true
    }

    fun removePreference(preference: Preference): Boolean = preferences.remove(preference)
    fun removeAll() = preferences.clear()
    val preferenceCount: Int get() = preferences.size
    fun getPreference(index: Int): Preference = preferences[index]
    fun findPreference(key: CharSequence): Preference? = preferences.firstOrNull { it.key == key }
}

class PreferenceScreen @JvmOverloads constructor(context: Context? = null) : PreferenceGroup(context)
