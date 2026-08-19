package tsunagu.repository

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.protobuf.ProtoBuf
import okhttp3.OkHttpClient
import okhttp3.Request
import java.util.zip.GZIPInputStream

class RepositoryFetchException(message: String) : Exception(message)

@OptIn(ExperimentalSerializationApi::class)
object RepositoryClient {
    private val client = OkHttpClient()
    private val json = Json { ignoreUnknownKeys = true }

    fun fetchIndex(indexUrl: String): List<ParsedExtension> {
        val request = Request.Builder().url(indexUrl).build()
        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            response.close()
            throw RepositoryFetchException("failed to fetch $indexUrl: HTTP ${response.code}")
        }
        val rawBytes = response.body?.bytes()
            ?: throw RepositoryFetchException("empty response body from $indexUrl")

        val bytes = if (rawBytes.size >= 2 && rawBytes[0] == 0x1f.toByte() && rawBytes[1] == 0x8b.toByte()) {
            runCatching {
                GZIPInputStream(rawBytes.inputStream()).readBytes()
            }.getOrElse {
                throw RepositoryFetchException("failed to gunzip $indexUrl: ${it.message}")
            }
        } else {
            rawBytes
        }

        val apkBaseUrl = indexUrl.substringBeforeLast('/')

        val asJsonText = runCatching { bytes.toString(Charsets.UTF_8) }.getOrNull()
        if (asJsonText != null && asJsonText.trimStart().startsWith("[")) {
            val legacyExtensions = runCatching {
                json.decodeFromString(ListSerializer(LegacyRepoExtension.serializer()), asJsonText)
            }.getOrNull()
            if (legacyExtensions != null) {
                return legacyExtensions.toParsedExtensions(apkBaseUrl)
            }
        }

        val index = runCatching {
            ProtoBuf.decodeFromByteArray(RepoIndex.serializer(), bytes)
        }.getOrElse {
            throw RepositoryFetchException("could not parse $indexUrl as JSON or protobuf index: ${it.message}")
        }

        if (index.extensionList == null && index.extensionListUrl != null) {
            throw RepositoryFetchException(
                "index at $indexUrl references an external extensionListUrl (${index.extensionListUrl}); " +
                    "paginated repo indices are not supported yet"
            )
        }

        return index.toParsedExtensions(apkBaseUrl)
    }
}
