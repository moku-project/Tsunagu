package tsunagu.novel

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.graalvm.polyglot.Value
import org.graalvm.polyglot.proxy.ProxyArray
import org.graalvm.polyglot.proxy.ProxyExecutable
import org.graalvm.polyglot.proxy.ProxyInstantiable
import org.graalvm.polyglot.proxy.ProxyObject
import org.jsoup.Jsoup
import org.jsoup.nodes.DataNode
import org.jsoup.nodes.Element
import org.jsoup.nodes.Node
import org.jsoup.nodes.TextNode

object NovelJsBridge {

    private val client = OkHttpClient()

    const val REQUIRE_GLUE = """
        function __require(name) {
            var mod = __hostRequire(name);
            if (mod === null || mod === undefined) {
                throw new Error("novel plugin requires unsupported module: " + name);
            }
            return mod;
        }
    """

    fun requireFn(storageNamespace: String, wrapPromise: Value): ProxyExecutable = ProxyExecutable { args ->
        when (args[0].asString()) {
            "cheerio" -> cheerioModule()
            "htmlparser2" -> htmlparser2Module()
            "@libs/fetch" -> fetchModule(wrapPromise)
            "@libs/novelStatus" -> novelStatusModule()
            "@libs/isAbsoluteUrl" -> isAbsoluteUrlModule()
            "@libs/defaultCover" -> defaultCoverModule()
            "@libs/utils" -> utilsModule()
            "@libs/filterInputs" -> filterInputsModule()
            "@libs/storage" -> storageModule(storageNamespace)
            else -> null
        }
    }

    private fun cheerioModule(): ProxyObject = ProxyObject.fromMap(mapOf(
        "load" to ProxyExecutable { args ->
            val doc = Jsoup.parse(args[0].asString())
            ProxyExecutable { selArgs -> CheerioSelection(doc.select(selArgs[0].asString())) }
        },
    ))

    private val VOID_ELEMENTS = setOf(
        "area", "base", "br", "col", "embed", "hr", "img", "input",
        "link", "meta", "param", "source", "track", "wbr",
    )

    private fun htmlparser2Module(): ProxyObject = ProxyObject.fromMap(mapOf(
        "Parser" to ProxyInstantiable { args -> parserInstance(args.getOrNull(0)) },
    ))

    private fun parserInstance(options: Value?): ProxyObject {
        val onopentag = options?.getMember("onopentag")?.takeIf { it.canExecute() }
        val ontext = options?.getMember("ontext")?.takeIf { it.canExecute() }
        val onclosetag = options?.getMember("onclosetag")?.takeIf { it.canExecute() }
        val onend = options?.getMember("onend")?.takeIf { it.canExecute() }

        val buffer = StringBuilder()

        return ProxyObject.fromMap(mapOf(
            "write" to ProxyExecutable { wargs -> buffer.append(wargs[0].asString()); null },
            "end" to ProxyExecutable {
                val doc = Jsoup.parseBodyFragment(buffer.toString())
                doc.body().childNodes().toList().forEach { node ->
                    walkHtmlNode(node, onopentag, ontext, onclosetag)
                }
                onend?.execute()
                null
            },
            "isVoidElement" to ProxyExecutable { iargs ->
                VOID_ELEMENTS.contains(iargs[0].asString().lowercase())
            },
        ))
    }

    private fun walkHtmlNode(node: Node, onopentag: Value?, ontext: Value?, onclosetag: Value?) {
        when (node) {
            is Element -> {
                val attribs = ProxyObject.fromMap(node.attributes().associate { it.key to it.value })
                onopentag?.execute(node.tagName(), attribs)
                node.childNodes().toList().forEach { walkHtmlNode(it, onopentag, ontext, onclosetag) }
                onclosetag?.execute(node.tagName())
            }
            is TextNode -> {
                ontext?.execute(node.wholeText)
            }
            is DataNode -> {
                ontext?.execute(node.wholeData)
            }
            else -> {}
        }
    }

    private fun fetchModule(wrapPromise: Value): ProxyObject = ProxyObject.fromMap(mapOf(
        "fetchApi" to ProxyExecutable { args -> wrapPromise.execute(doFetch(args)) },
        "fetchText" to ProxyExecutable { args ->
            val text = try { rawFetch(args).body } catch (e: Exception) { "" }
            wrapPromise.execute(text)
        },
    ))

    private fun doFetch(args: Array<Value>): ProxyObject {
        val raw = rawFetch(args)
        return ProxyObject.fromMap(mapOf(
            "ok" to raw.ok,
            "status" to raw.status,
            "url" to raw.url,
            "json" to ProxyExecutable { Json.parseToJsonElement(raw.body).toJsValue() },
            "text" to ProxyExecutable { raw.body },
        ))
    }

    private data class RawResponse(val ok: Boolean, val status: Int, val url: String, val body: String)

    private fun rawFetch(args: Array<Value>): RawResponse {
        val url = args[0].asString()
        val init = args.getOrNull(1)?.takeIf { !it.isNull }

        val method = init?.getMember("method")?.takeIf { !it.isNull }?.asString() ?: "GET"
        val bodyValue = init?.getMember("body")?.takeIf { !it.isNull }
        val bodyStr = when {
            bodyValue == null -> null
            bodyValue.isString -> bodyValue.asString()
            bodyValue.hasMember("_entries") -> {
                val entries = bodyValue.getMember("_entries")
                (0 until entries.arraySize).joinToString("&") { i ->
                    val pair = entries.getArrayElement(i)
                    val k = java.net.URLEncoder.encode(pair.getArrayElement(0).asString(), "UTF-8")
                    val v = java.net.URLEncoder.encode(pair.getArrayElement(1).asString(), "UTF-8")
                    "$k=$v"
                }
            }
            else -> null
        }

        val builder = Request.Builder().url(url)
        init?.getMember("headers")?.takeIf { !it.isNull && it.hasMembers() }?.let { headers ->
            headers.memberKeys.forEach { key -> builder.addHeader(key, headers.getMember(key).asString()) }
        }
        if (bodyStr != null && method != "GET" && method != "HEAD") {
            builder.method(method, bodyStr.toRequestBody())
        } else {
            builder.method(method, null)
        }

        client.newCall(builder.build()).execute().use { resp ->
            return RawResponse(
                ok = resp.isSuccessful,
                status = resp.code,
                url = resp.request.url.toString(),
                body = resp.body?.string() ?: "",
            )
        }
    }

    private fun JsonElement.toJsValue(): Any? = when (this) {
        is JsonNull -> null
        is JsonPrimitive -> when {
            isString -> content
            content == "true" -> true
            content == "false" -> false
            else -> content.toDoubleOrNull() ?: content
        }
        is JsonArray -> ProxyArray.fromList(map { it.toJsValue() })
        is JsonObject -> ProxyObject.fromMap(entries.associate { (k, v) -> k to v.toJsValue() })
    }

    private fun novelStatusModule(): ProxyObject = ProxyObject.fromMap(mapOf(
        "NovelStatus" to ProxyObject.fromMap(mapOf(
            "Unknown" to "Unknown", "Ongoing" to "Ongoing", "Completed" to "Completed",
            "Licensed" to "Licensed", "PublishingFinished" to "Publishing Finished",
            "Cancelled" to "Cancelled", "OnHiatus" to "On Hiatus",
        )),
    ))

    private fun isAbsoluteUrlModule(): ProxyObject = ProxyObject.fromMap(mapOf(
        "isUrlAbsolute" to ProxyExecutable { args ->
            args[0].asString().let { it.startsWith("http://") || it.startsWith("https://") }
        },
    ))

    private fun defaultCoverModule(): ProxyObject = ProxyObject.fromMap(mapOf("defaultCover" to ""))

    private fun utilsModule(): ProxyObject = ProxyObject.fromMap(mapOf(
        "utf8ToBytes" to ProxyExecutable { args -> args[0].asString().toByteArray(Charsets.UTF_8) },
        "bytesToUtf8" to ProxyExecutable { args -> String(args[0].`as`(ByteArray::class.java), Charsets.UTF_8) },
    ))

    private fun storageModule(storageNamespace: String): ProxyObject = ProxyObject.fromMap(mapOf(
        "storage" to ProxyObject.fromMap(mapOf(
            "get" to ProxyExecutable { args -> PluginStorage.get(storageNamespace, args[0].asString()) },
            "set" to ProxyExecutable { args ->
                PluginStorage.set(storageNamespace, args[0].asString(), args.getOrNull(1)?.jsValueToKotlin())
                null
            },
            "delete" to ProxyExecutable { args -> PluginStorage.delete(storageNamespace, args[0].asString()); null },
        )),
    ))

    private fun Value.jsValueToKotlin(): Any? = when {
        isNull -> null
        isBoolean -> asBoolean()
        isNumber -> asDouble()
        isString -> asString()
        else -> toString()
    }

    private fun filterInputsModule(): ProxyObject = ProxyObject.fromMap(mapOf(
        "FilterTypes" to ProxyObject.fromMap(mapOf(
            "TextInput" to "Text",
            "Picker" to "Picker",
            "CheckboxGroup" to "Checkbox",
            "ExcludableCheckboxGroup" to "XCheckbox",
            "Switch" to "Switch",
        )),
    ))
}