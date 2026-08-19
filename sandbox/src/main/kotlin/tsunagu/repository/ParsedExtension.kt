package tsunagu.repository

data class ParsedExtension(
    val name: String,
    val packageName: String,
    val apkUrl: String,
    val jarUrl: String? = null,
    val iconUrl: String?,
    val versionName: String,
    val lang: String,
)
