package tsunagu.repository

import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.Serializable
import kotlinx.serialization.protobuf.ProtoNumber

@OptIn(ExperimentalSerializationApi::class)
@Serializable
data class RepoIndex(
    @ProtoNumber(1) val name: String,
    @ProtoNumber(2) val badgeLabel: String,
    @ProtoNumber(3) val signingKey: String,
    @ProtoNumber(4) val contact: Contact,
    @ProtoNumber(101) val extensionList: ExtensionList? = null,
    @ProtoNumber(102) val extensionListUrl: String? = null,
) {
    @Serializable
    data class Contact(
        @ProtoNumber(1) val website: String,
        @ProtoNumber(2) val discord: String? = null,
    )

    @Serializable
    data class ExtensionList(
        @ProtoNumber(1) val extensions: List<Extension>,
    )

    @Serializable
    data class Extension(
        @ProtoNumber(1) val name: String,
        @ProtoNumber(2) val packageName: String,
        @ProtoNumber(3) val resources: Resources,
        @ProtoNumber(4) val extensionLib: String,
        @ProtoNumber(5) val versionCode: Long,
        @ProtoNumber(6) val versionName: String,
        @ProtoNumber(7) val contentWarning: Int = 0,
        @ProtoNumber(8) val sources: List<Source> = emptyList(),
    )

    @Serializable
    data class Resources(
        @ProtoNumber(1) val apkUrl: String,
        @ProtoNumber(2) val iconUrl: String,
        @ProtoNumber(501) val jarUrl: String? = null,
    )

    @Serializable
    data class Source(
        @ProtoNumber(1) val id: Long,
        @ProtoNumber(2) val name: String,
        @ProtoNumber(3) val language: String,
        @ProtoNumber(4) val homeUrl: String = "",
        @ProtoNumber(5) val mirrorUrls: List<String> = emptyList(),
        @ProtoNumber(7) val message: String? = null,
    )

    fun toParsedExtensions(baseRawUrl: String): List<ParsedExtension> =
        (extensionList?.extensions ?: emptyList()).map { ext ->
            val lang = ext.sources.map { it.language }.toSet().let { if (it.size == 1) it.first() else "all" }
            ParsedExtension(
                name = ext.name,
                packageName = ext.packageName,
                apkUrl = ext.resources.apkUrl,
                jarUrl = ext.resources.jarUrl,
                iconUrl = ext.resources.iconUrl,
                versionName = ext.versionName,
                lang = lang,
            )
        }
}

@Serializable
data class LegacyRepoExtension(
    val name: String,
    val pkg: String,
    val apk: String,
    val lang: String,
    val version: String,
)

fun List<LegacyRepoExtension>.toParsedExtensions(apkBaseUrl: String): List<ParsedExtension> =
    map { ext ->
        ParsedExtension(
            name = ext.name,
            packageName = ext.pkg,
            apkUrl = "$apkBaseUrl/${ext.apk}",
            jarUrl = null,
            iconUrl = null,
            versionName = ext.version,
            lang = ext.lang,
        )
    }
