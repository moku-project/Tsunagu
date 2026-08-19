package tsunagu.repository

import java.util.concurrent.ConcurrentHashMap

data class RepositoryEntry(
    val id: String,
    val indexUrl: String,
    val extensions: List<ParsedExtension>,
)

class RepositoryNotFoundException(message: String) : Exception(message)
class ExtensionNotFoundInRepositoryException(message: String) : Exception(message)

class RepositoryRegistry {
    private val repositories = ConcurrentHashMap<String, RepositoryEntry>()

    fun add(indexUrl: String): RepositoryEntry {
        val extensions = RepositoryClient.fetchIndex(indexUrl)
        val entry = RepositoryEntry(
            id = indexUrl,
            indexUrl = indexUrl,
            extensions = extensions,
        )
        repositories[entry.id] = entry
        return entry
    }

    fun list(): List<RepositoryEntry> = repositories.values.toList()

    fun get(repositoryId: String): RepositoryEntry =
        repositories[repositoryId] ?: throw RepositoryNotFoundException("repository not found: $repositoryId")

    fun findExtension(repositoryId: String, extensionId: String): ParsedExtension {
        val repo = get(repositoryId)
        return repo.extensions.find { it.packageName == extensionId }
            ?: throw ExtensionNotFoundInRepositoryException(
                "extension $extensionId not found in repository $repositoryId"
            )
    }
}