package tsunagu.grpc

import android.app.Application
import android.content.SharedPreferences
import androidx.preference.CheckBoxPreference
import androidx.preference.EditTextPreference
import androidx.preference.ListPreference
import androidx.preference.MultiSelectListPreference
import androidx.preference.Preference
import androidx.preference.PreferenceGroup
import androidx.preference.PreferenceScreen
import androidx.preference.SwitchPreferenceCompat
import eu.kanade.tachiyomi.animesource.AnimeSource
import eu.kanade.tachiyomi.animesource.ConfigurableAnimeSource
import eu.kanade.tachiyomi.source.ConfigurableSource
import eu.kanade.tachiyomi.source.Source
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import org.koin.core.context.GlobalContext
import sandbox.v1.Sandbox

object SourcePrefSerde {

    private val json = Json { ignoreUnknownKeys = true }

    private fun sourceId(source: Any): Long? = when (source) {
        is Source -> source.id
        is AnimeSource -> source.id
        else -> null
    }

    fun prefsFor(source: Any): SharedPreferences? {
        val id = sourceId(source) ?: return null
        val app = GlobalContext.get().get<Application>()
        return app.getSharedPreferences("source_$id", 0)
    }

    private fun collectScreen(source: Any): List<Preference> {
        val screen = PreferenceScreen(GlobalContext.get().get<Application>())
        when (source) {
            is ConfigurableSource -> source.setupPreferenceScreen(screen)
            is ConfigurableAnimeSource -> source.setupPreferenceScreen(screen)
            else -> return emptyList()
        }
        val out = mutableListOf<Preference>()
        fun walk(group: PreferenceGroup) {
            for (p in group.preferences) {
                if (p is PreferenceGroup) walk(p) else out.add(p)
            }
        }
        walk(screen)
        return out
    }

    fun harvest(source: Any): List<Sandbox.SourcePreference> {
        val prefs = prefsFor(source)
        return collectScreen(source).mapNotNull { p ->
            val key = p.key ?: return@mapNotNull null
            val b = Sandbox.SourcePreference.newBuilder()
                .setKey(key)
                .setTitle(p.title?.toString().orEmpty())
                .setSummary(p.summary?.toString().orEmpty())
            when (p) {
                is SwitchPreferenceCompat, is CheckBoxPreference -> {
                    val def = (p.defaultValue as? Boolean) ?: false
                    val cur = prefs?.getBoolean(key, def) ?: def
                    b.setType("switch").setDefaultValue(def.toString()).setCurrentValue(cur.toString())
                }
                is MultiSelectListPreference -> {
                    b.setType("multiselect")
                    b.addAllEntries(p.entries?.map { it.toString() } ?: emptyList())
                    b.addAllEntryValues(p.entryValues?.map { it.toString() } ?: emptyList())
                    @Suppress("UNCHECKED_CAST")
                    val def = (p.defaultValue as? Set<String>) ?: emptySet()
                    val cur = prefs?.getStringSet(key, def) ?: def
                    b.setDefaultValue(encodeSet(def)).setCurrentValue(encodeSet(cur))
                }
                is ListPreference -> {
                    b.setType("list")
                    b.addAllEntries(p.entries?.map { it.toString() } ?: emptyList())
                    b.addAllEntryValues(p.entryValues?.map { it.toString() } ?: emptyList())
                    val def = p.defaultValue?.toString().orEmpty()
                    val cur = prefs?.getString(key, def) ?: def
                    b.setDefaultValue(def).setCurrentValue(cur)
                }
                is EditTextPreference -> {
                    b.setType("edittext")
                    val def = p.defaultValue?.toString().orEmpty()
                    val cur = prefs?.getString(key, def) ?: def
                    b.setDefaultValue(def).setCurrentValue(cur)
                }
                else -> return@mapNotNull null
            }
            b.build()
        }
    }

    fun apply(source: Any, key: String, value: String) {
        val prefs = prefsFor(source) ?: throw IllegalStateException("source has no id / preferences")
        val decl = collectScreen(source).firstOrNull { it.key == key }
            ?: throw IllegalArgumentException("unknown preference key: $key")
        val editor = prefs.edit()
        when (decl) {
            is SwitchPreferenceCompat, is CheckBoxPreference ->
                editor.putBoolean(key, value.trim().equals("true", ignoreCase = true))
            is MultiSelectListPreference ->
                editor.putStringSet(key, decodeSet(value))
            is ListPreference, is EditTextPreference ->
                editor.putString(key, value)
            else -> throw IllegalArgumentException("preference $key is not settable")
        }
        editor.apply()
    }

    private fun encodeSet(s: Set<String>): String =
        json.encodeToString(ListSerializer(String.serializer()), s.toList())

    private fun decodeSet(v: String): Set<String> =
        runCatching { json.decodeFromString(ListSerializer(String.serializer()), v).toSet() }
            .getOrElse { setOf(v) }
}
