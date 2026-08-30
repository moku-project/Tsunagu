package tsunagu.novel

import org.graalvm.polyglot.Context
import org.graalvm.polyglot.HostAccess
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class DayjsShimTest {

    private fun ctx(): Context {
        val c = Context.newBuilder("js", "regex")
            .allowHostAccess(HostAccess.ALL)
            .allowHostClassLookup { false }
            .build()
        c.getBindings("js").putMember(
            "__hostRequire",
            NovelJsBridge.requireFn("test", c.eval("js", "(function(v){return Promise.resolve(v);})")),
        )
        c.eval("js", NovelJsBridge.REQUIRE_GLUE)
        return c
    }

    @Test
    fun subtractAndFormatLL() {
        ctx().use { c ->
            val out = c.eval(
                "js",
                """
                var dayjs = __require('dayjs');
                dayjs('2024-08-04').subtract(3, 'days').format('LL');
                """,
            ).asString()
            assertEquals("August 1, 2024", out)
        }
    }

    @Test
    fun relativeUnitsAliasPlural() {
        ctx().use { c ->
            val out = c.eval(
                "js",
                "var d=__require('dayjs'); d('2024-01-15T12:00:00Z').subtract(2,'hours').subtract(30,'minute').format('YYYY-MM-DD HH:mm');",
            ).asString()
            assertEquals("2024-01-15 09:30", out)
        }
    }

    @Test
    fun invalidDateSentinel() {
        ctx().use { c ->
            val out = c.eval(
                "js",
                "var d=__require('dayjs'); d('not a date at all').format('LL');",
            ).asString()
            assertEquals("Invalid Date", out)
        }
    }

    @Test
    fun importDefaultInteropAndNow() {
        ctx().use { c ->

            val ok = c.eval(
                "js",
                """
                var raw = __require('dayjs');
                var o = raw && raw.__esModule ? raw : { default: raw };
                var n = (0, o.default)();
                typeof n.format('LL') === 'string' && n.isValid();
                """,
            ).asBoolean()
            assertTrue(ok)
        }
    }
}
