package eu.kanade.tachiyomi.animesource

import androidx.preference.PreferenceScreen

@Suppress("unused")
interface ConfigurableAnimeSource {

    fun setupPreferenceScreen(screen: PreferenceScreen)

}
