package tsunagu.source

import eu.kanade.tachiyomi.source.Source
import tsunagu.registry.ExtensionRegistry

object GetSource {
    @Volatile
    private var registry: ExtensionRegistry? = null

    fun bind(extensionRegistry: ExtensionRegistry) {
        registry = extensionRegistry
    }

    fun unregisterAllSources() {
        registry?.invalidateAll()
    }
}

typealias LoadedSource = Source
