package tsunagu.novel

import org.graalvm.polyglot.Value

internal fun Value.str(key: String): String? {
    val member = getMember(key) ?: return null
    return if (member.isNull) null else member.asString()
}

internal fun Value.dbl(key: String): Double? {
    val member = getMember(key) ?: return null
    return if (member.isNull || !member.isNumber) null else member.asDouble()
}
