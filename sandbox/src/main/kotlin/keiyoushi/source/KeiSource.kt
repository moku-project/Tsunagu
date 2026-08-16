package keiyoushi.source

import eu.kanade.tachiyomi.source.model.SManga
import eu.kanade.tachiyomi.source.online.HttpSource
import okhttp3.HttpUrl

abstract class KeiSource : HttpSource() {
    open suspend fun getMangaByUrl(url: HttpUrl): SManga? = null
}
