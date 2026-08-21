package tsunagu.novel

import org.graalvm.polyglot.Context
import org.graalvm.polyglot.HostAccess
import org.graalvm.polyglot.PolyglotException
import tsunagu.loader.ContentType
import tsunagu.loader.ExtensionLoadException
import tsunagu.loader.LoadedExtension
import java.io.File

object NovelPluginLoader {

    private const val FORM_DATA_GLUE = """
        function FormData() {
            this._entries = [];
        }
        FormData.prototype.append = function(k, v) { this._entries.push([k, v]); };
    """

    private const val URL_SEARCH_PARAMS_GLUE = """
        function URLSearchParams() {
            this._entries = [];
        }
        URLSearchParams.prototype.append = function(k, v) { this._entries.push([k, v]); };
        URLSearchParams.prototype.toString = function() {
            return this._entries.map(function(p) {
                return encodeURIComponent(p[0]) + '=' + encodeURIComponent(p[1]);
            }).join('&');
        };
    """

    fun load(file: File): LoadedExtension {
        val rawCode = file.readText()

        val context = Context.newBuilder("js")
            .allowHostAccess(HostAccess.ALL)
            .allowHostClassLookup { false }
            .build()

        context.getBindings("js").putMember("__hostRequire", NovelJsBridge.requireFn())
        context.eval("js", NovelJsBridge.REQUIRE_GLUE)
        context.eval("js", FORM_DATA_GLUE)
        context.eval("js", URL_SEARCH_PARAMS_GLUE)

        val wrapped = """
            (function() {
              var module = { exports: {} };
              var exports = module.exports;
              (function(require, module, exports) {
              $rawCode
              })(__require, module, exports);
              return module.exports.default;
            })()
        """.trimIndent()

        val pluginValue = try {
            context.eval("js", wrapped)
        } catch (e: PolyglotException) {
            context.close(true)
            throw ExtensionLoadException("failed to evaluate novel plugin ${file.name}: ${e.message}")
        }

        if (pluginValue == null || pluginValue.isNull) {
            context.close(true)
            throw ExtensionLoadException("novel plugin ${file.name} did not export a default plugin object")
        }

        val id = pluginValue.str("id") ?: run {
            context.close(true)
            throw ExtensionLoadException("novel plugin ${file.name} missing required field: id")
        }
        val name = pluginValue.str("name") ?: id
        val site = pluginValue.str("site") ?: ""
        val lang = pluginValue.str("lang") ?: ""
        val version = pluginValue.str("version") ?: "0.0.0"

        val plugin = NovelPlugin(id, name, site, lang, version, context, pluginValue)

        return LoadedExtension(
            packageName = id,
            source = plugin,
            classLoader = NovelPluginLoader::class.java.classLoader,
            contentType = ContentType.NOVEL,
        )
    }
}
