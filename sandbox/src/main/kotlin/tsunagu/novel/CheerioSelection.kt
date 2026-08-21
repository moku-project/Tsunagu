package tsunagu.novel

import org.graalvm.polyglot.Value
import org.graalvm.polyglot.proxy.ProxyArray
import org.graalvm.polyglot.proxy.ProxyExecutable
import org.graalvm.polyglot.proxy.ProxyObject
import org.jsoup.select.Elements

class CheerioSelection(private val elements: Elements) : ProxyArray, ProxyObject {

    override fun get(index: Long): Any =
        CheerioSelection(Elements(elements[index.toInt()]))

    override fun set(index: Long, value: Value?) {
        throw UnsupportedOperationException("CheerioSelection is read-only")
    }

    override fun getSize(): Long = elements.size.toLong()

    override fun getMember(key: String): Any? = when (key) {
        "text" -> ProxyExecutable { elements.text() }
        "html" -> ProxyExecutable { elements.html() }
        "attr" -> ProxyExecutable { args -> elements.attr(args[0].asString()) }
        "find" -> ProxyExecutable { args -> CheerioSelection(elements.select(args[0].asString())) }
        "each" -> ProxyExecutable { args ->
            val cb = args[0]
            elements.forEachIndexed { i, el -> cb.execute(i.toLong(), CheerioSelection(Elements(el))) }
            this
        }
        "eq" -> ProxyExecutable { args ->
            val i = args[0].asInt()
            CheerioSelection(if (i in elements.indices) Elements(elements[i]) else Elements())
        }
        "first" -> ProxyExecutable {
            CheerioSelection(if (elements.isNotEmpty()) Elements(elements[0]) else Elements())
        }
        "last" -> ProxyExecutable {
            CheerioSelection(if (elements.isNotEmpty()) Elements(elements[elements.size - 1]) else Elements())
        }
        "length" -> elements.size.toLong()
        else -> null
    }

    override fun getMemberKeys(): Any =
        ProxyArray.fromArray("text", "html", "attr", "find", "each", "eq", "first", "last", "length")

    override fun hasMember(key: String): Boolean = getMember(key) != null

    override fun putMember(key: String, value: Value?) {

    }

    override fun removeMember(key: String): Boolean = false
}
