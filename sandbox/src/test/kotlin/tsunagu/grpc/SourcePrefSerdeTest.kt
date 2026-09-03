package tsunagu.grpc

import android.app.Application
import android.content.FileBackedSharedPreferences
import androidx.preference.ListPreference
import androidx.preference.MultiSelectListPreference
import androidx.preference.PreferenceScreen
import androidx.preference.SwitchPreferenceCompat
import eu.kanade.tachiyomi.source.ConfigurableSource
import eu.kanade.tachiyomi.source.model.FilterList
import eu.kanade.tachiyomi.source.model.MangasPage
import eu.kanade.tachiyomi.source.model.Page
import eu.kanade.tachiyomi.source.model.SChapter
import eu.kanade.tachiyomi.source.model.SManga
import eu.kanade.tachiyomi.source.model.SMangaUpdate
import eu.kanade.tachiyomi.source.Source
import org.koin.core.context.GlobalContext
import org.koin.core.context.startKoin
import org.koin.core.context.stopKoin
import org.koin.dsl.module
import java.io.File
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

private class FakeSource(private val fakeId: Long) : Source, ConfigurableSource {
    override val id: Long get() = fakeId
    override val name = "Fake"
    override val supportsLatest = false
    override suspend fun getPopularManga(page: Int) = MangasPage(emptyList(), false)
    override suspend fun getLatestUpdates(page: Int) = MangasPage(emptyList(), false)
    override suspend fun getSearchManga(page: Int, query: String, filters: FilterList) = MangasPage(emptyList(), false)
    override suspend fun getMangaUpdate(manga: SManga, chapters: List<SChapter>, fetchDetails: Boolean, fetchChapters: Boolean): SMangaUpdate = throw UnsupportedOperationException()
    override suspend fun getPageList(chapter: SChapter): List<Page> = emptyList()

    override fun setupPreferenceScreen(screen: PreferenceScreen) {
        screen.addPreference(
            ListPreference(screen.context).apply {
                key = "lang"
                title = "Language"
                entries = arrayOf("English", "Japanese")
                entryValues = arrayOf("en", "ja")
                setDefaultValue("en")
            },
        )
        screen.addPreference(
            SwitchPreferenceCompat(screen.context).apply {
                key = "nsfw"
                title = "Show NSFW"
                setDefaultValue(false)
            },
        )
        screen.addPreference(
            MultiSelectListPreference(screen.context).apply {
                key = "ratings"
                title = "Content ratings"
                entries = arrayOf("Safe", "Suggestive", "Erotica", "Pornographic")
                entryValues = arrayOf("safe", "suggestive", "erotica", "pornographic")
                setDefaultValue(setOf("safe", "suggestive", "erotica"))
            },
        )
    }
}

class SourcePrefSerdeTest {

    private lateinit var tmp: File

    @BeforeTest
    fun setup() {
        tmp = File.createTempFile("prefs", "").apply { delete(); mkdirs() }
        Application.prefsRoot = tmp
        startKoin { modules(module { single { Application() } }) }
    }

    @AfterTest
    fun teardown() {
        stopKoin()
        tmp.deleteRecursively()
    }

    @Test
    fun harvestReportsDefaultsThenReflectsWrites() {
        val src = FakeSource(42L)

        val before = SourcePrefSerde.harvest(src).associateBy { it.key }
        assertEquals(3, before.size)
        assertEquals("list", before["lang"]!!.type)
        assertEquals("en", before["lang"]!!.currentValue)
        assertEquals(listOf("en", "ja"), before["lang"]!!.entryValuesList)
        assertEquals("switch", before["nsfw"]!!.type)
        assertEquals("false", before["nsfw"]!!.currentValue)
        assertEquals("multiselect", before["ratings"]!!.type)
        assertTrue(before["ratings"]!!.currentValue.contains("erotica"))
        assertTrue(!before["ratings"]!!.currentValue.contains("pornographic"))

        SourcePrefSerde.apply(src, "lang", "ja")
        SourcePrefSerde.apply(src, "nsfw", "true")
        SourcePrefSerde.apply(src, "ratings", """["safe","suggestive","erotica","pornographic"]""")

        val after = SourcePrefSerde.harvest(src).associateBy { it.key }
        assertEquals("ja", after["lang"]!!.currentValue)
        assertEquals("true", after["nsfw"]!!.currentValue)
        assertTrue(after["ratings"]!!.currentValue.contains("pornographic"))

        // persisted to disk and visible to a fresh prefs instance
        val fresh = FileBackedSharedPreferences(File(tmp, "source_42.json"))
        assertEquals("ja", fresh.getString("lang", null))
        assertEquals(true, fresh.getBoolean("nsfw", false))
        assertTrue(fresh.getStringSet("ratings", emptySet())!!.contains("pornographic"))
    }

    @Test
    fun applyRejectsUnknownKey() {
        val src = FakeSource(7L)
        val ex = runCatching { SourcePrefSerde.apply(src, "nope", "x") }.exceptionOrNull()
        assertTrue(ex is IllegalArgumentException)
    }
}
