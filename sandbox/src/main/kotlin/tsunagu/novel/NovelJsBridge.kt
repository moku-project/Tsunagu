package tsunagu.novel

import eu.kanade.tachiyomi.network.buildSandboxDns
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

    /** The client every source uses, so plugins share cookies, Cloudflare handling and solved User-Agents. */
    internal val client: OkHttpClient by lazy {
        org.koin.core.context.GlobalContext.getOrNull()?.get<eu.kanade.tachiyomi.network.NetworkHelper>()?.client
            ?: OkHttpClient.Builder()
                .dns(buildSandboxDns(System.getenv("SANDBOX_DOH") ?: "off"))
                .build()
    }

    const val REQUIRE_GLUE = """
        function __require(name) {
            var mod = __hostRequire(name);
            if (mod === null || mod === undefined) {
                throw new Error("novel plugin requires unsupported module: " + name);
            }
            return name === "@libs/aes" ? __wrapAes(mod) : mod;
        }
        // Host ciphers return Java byte[]; plugins expect Uint8Array like noble-ciphers.
        function __wrapAes(m) {
            function u8(b) { return Uint8Array.from(b, function (x) { return x & 255; }); }
            var out = {};
            ["gcm", "cbc", "ctr", "ecb"].forEach(function (k) {
                out[k] = function (a, b, c) {
                    var cipher = m[k](a, b, c);
                    return {
                        encrypt: function (d) { return u8(cipher.encrypt(d)); },
                        decrypt: function (d) { return u8(cipher.decrypt(d)); },
                    };
                };
            });
            return out;
        }
    """

    // Web globals plugins expect from React Native that GraalJS lacks. Calls run synchronously,
    // so timers sleep on the host and fire as a microtask.
    const val WEB_GLUE = """
        (function (g) {
            function enc(s) { return encodeURIComponent(s).replace(/%20/g, '+'); }
            function dec(s) { return decodeURIComponent(String(s).replace(/\+/g, ' ')); }

            function URLSearchParams(init) {
                var self = this;
                this._entries = [];
                if (init == null) return;
                if (typeof init === 'string') {
                    init.replace(/^\?/, '').split('&').forEach(function (pair) {
                        if (!pair) return;
                        var i = pair.indexOf('=');
                        self._entries.push(i >= 0 ? [dec(pair.slice(0, i)), dec(pair.slice(i + 1))] : [dec(pair), '']);
                    });
                } else if (init instanceof URLSearchParams) {
                    this._entries = init._entries.map(function (p) { return [p[0], p[1]]; });
                } else if (Array.isArray(init)) {
                    init.forEach(function (p) { self._entries.push([String(p[0]), String(p[1])]); });
                } else if (typeof init === 'object') {
                    Object.keys(init).forEach(function (k) { self._entries.push([k, String(init[k])]); });
                }
            }
            URLSearchParams.prototype.append = function (k, v) { this._entries.push([String(k), String(v)]); };
            URLSearchParams.prototype.set = function (k, v) {
                k = String(k);
                var i = this._entries.findIndex(function (p) { return p[0] === k; });
                this.delete(k);
                if (i < 0 || i > this._entries.length) i = this._entries.length;
                this._entries.splice(i, 0, [k, String(v)]);
            };
            URLSearchParams.prototype.get = function (k) {
                var e = this._entries.find(function (p) { return p[0] === String(k); });
                return e ? e[1] : null;
            };
            URLSearchParams.prototype.getAll = function (k) {
                return this._entries.filter(function (p) { return p[0] === String(k); }).map(function (p) { return p[1]; });
            };
            URLSearchParams.prototype.has = function (k) { return this.get(k) !== null; };
            URLSearchParams.prototype.delete = function (k) { this._entries = this._entries.filter(function (p) { return p[0] !== String(k); }); };
            URLSearchParams.prototype.forEach = function (cb, thisArg) {
                var self = this;
                this._entries.forEach(function (p) { cb.call(thisArg, p[1], p[0], self); });
            };
            URLSearchParams.prototype.entries = function () { return this._entries.map(function (p) { return [p[0], p[1]]; })[Symbol.iterator](); };
            URLSearchParams.prototype.keys = function () { return this._entries.map(function (p) { return p[0]; })[Symbol.iterator](); };
            URLSearchParams.prototype.values = function () { return this._entries.map(function (p) { return p[1]; })[Symbol.iterator](); };
            URLSearchParams.prototype[Symbol.iterator] = URLSearchParams.prototype.entries;
            URLSearchParams.prototype.toString = function () {
                return this._entries.map(function (p) { return enc(p[0]) + '=' + enc(p[1]); }).join('&');
            };
            Object.defineProperty(URLSearchParams.prototype, 'size', { get: function () { return this._entries.length; } });

            var ABS = /^([a-zA-Z][a-zA-Z0-9+.-]*):(?:\/\/(?:([^\/?#@]*)@)?([^\/?#:]*|\[[^\]]*\])(?::(\d*))?)?([^?#]*)(\?[^#]*)?(#.*)?$/;
            var DEFAULT_PORT = { 'http:': '80', 'https:': '443', 'ws:': '80', 'wss:': '443', 'ftp:': '21' };

            function removeDots(path) {
                var out = [];
                var segs = path.split('/');
                for (var i = 0; i < segs.length; i++) {
                    var s = segs[i];
                    if (s === '..') { if (out.length > 1) out.pop(); if (i === segs.length - 1) out.push(''); }
                    else if (s === '.') { if (i === segs.length - 1) out.push(''); }
                    else out.push(s);
                }
                return out.join('/');
            }

            function parse(input) {
                var m = ABS.exec(String(input).trim());
                if (!m) return null;
                var protocol = m[1].toLowerCase() + ':';
                var special = protocol in DEFAULT_PORT || protocol === 'file:';
                var path = m[5] || '';
                if (special && m[3] !== undefined) path = removeDots(path.replace(/\\/g, '/') || '/');
                var port = m[4] || '';
                if (DEFAULT_PORT[protocol] === port) port = '';
                return {
                    protocol: protocol,
                    hasAuthority: m[3] !== undefined,
                    username: m[2] ? m[2].split(':')[0] : '',
                    password: m[2] && m[2].indexOf(':') >= 0 ? m[2].slice(m[2].indexOf(':') + 1) : '',
                    hostname: (m[3] || '').toLowerCase(),
                    port: port,
                    pathname: path.replace(/ /g, '%20'),
                    search: m[6] && m[6].length > 1 ? m[6] : '',
                    hash: m[7] && m[7].length > 1 ? m[7] : '',
                };
            }

            function resolve(ref, base) {
                ref = String(ref).trim();
                if (ABS.test(ref) && /^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(ref)) return parse(ref);
                var b = parse(base);
                if (!b) return null;
                var r = { protocol: b.protocol, hasAuthority: b.hasAuthority, username: b.username, password: b.password,
                          hostname: b.hostname, port: b.port, pathname: b.pathname, search: b.search, hash: '' };
                var m = /^([^?#]*)(\?[^#]*)?(#.*)?$/.exec(ref);
                var path = m[1], search = m[2], hash = m[3];
                if (path.indexOf('//') === 0) return parse(b.protocol + ref);
                if (path === '') {
                    if (search !== undefined) r.search = search.length > 1 ? search : '';
                } else {
                    if (path.charAt(0) === '/') r.pathname = removeDots(path);
                    else r.pathname = removeDots(b.pathname.replace(/[^\/]*$/, '') + path);
                    r.search = search && search.length > 1 ? search : '';
                }
                r.pathname = r.pathname.replace(/ /g, '%20');
                r.hash = hash && hash.length > 1 ? hash : '';
                return r;
            }

            function URL(url, base) {
                var p = base === undefined || base === null ? parse(url) : resolve(url, base instanceof URL ? base.href : base);
                if (!p) throw new TypeError('Invalid URL: ' + url);
                this._p = p;
                this._sp = new URLSearchParams(p.search);
                var self = this;
                var sync = function () { var s = URLSearchParams.prototype.toString.call(self._sp); self._p.search = s ? '?' + s : ''; };
                ['append', 'set', 'delete'].forEach(function (k) {
                    self._sp[k] = function () { URLSearchParams.prototype[k].apply(self._sp, arguments); sync(); };
                });
            }
            function prop(name, get, set) { Object.defineProperty(URL.prototype, name, { get: get, set: set, enumerable: true }); }
            prop('protocol', function () { return this._p.protocol; }, function (v) { this._p.protocol = String(v).replace(/:?$/, ':'); });
            prop('username', function () { return this._p.username; });
            prop('password', function () { return this._p.password; });
            prop('hostname', function () { return this._p.hostname; }, function (v) { this._p.hostname = String(v); });
            prop('port', function () { return this._p.port; }, function (v) { this._p.port = String(v); });
            prop('host', function () { return this._p.hostname + (this._p.port ? ':' + this._p.port : ''); });
            prop('origin', function () { return this._p.hasAuthority ? this._p.protocol + '//' + this.host : 'null'; });
            prop('pathname', function () { return this._p.pathname; }, function (v) { v = String(v); this._p.pathname = v.charAt(0) === '/' ? v : '/' + v; });
            prop('search', function () { return this._p.search; }, function (v) {
                v = String(v).replace(/^\?/, '');
                this._p.search = v ? '?' + v : '';
                this._sp._entries = new URLSearchParams(v)._entries;
            });
            prop('hash', function () { return this._p.hash; }, function (v) { v = String(v).replace(/^#/, ''); this._p.hash = v ? '#' + v : ''; });
            prop('searchParams', function () { return this._sp; });
            prop('href', function () {
                var p = this._p;
                var auth = p.username ? p.username + (p.password ? ':' + p.password : '') + '@' : '';
                return p.protocol + (p.hasAuthority ? '//' + auth + this.host : '') + p.pathname + p.search + p.hash;
            }, function (v) { var n = new URL(v); this._p = n._p; this._sp._entries = n._sp._entries; });
            URL.prototype.toString = function () { return this.href; };
            URL.prototype.toJSON = function () { return this.href; };
            URL.canParse = function (u, b) { try { new URL(u, b); return true; } catch (e) { return false; } };

            function Headers(init) {
                var self = this;
                if (init == null) return;
                if (Array.isArray(init)) init.forEach(function (p) { self.append(p[0], p[1]); });
                else Object.keys(init).forEach(function (k) { self.append(k, init[k]); });
            }
            // Headers live as own enumerable props so fetch's init.headers handling reads them like a plain object.
            function hdef(name, fn) { Object.defineProperty(Headers.prototype, name, { value: fn, enumerable: false }); }
            hdef('_key', function (k) {
                k = String(k).toLowerCase();
                return Object.keys(this).find(function (x) { return x.toLowerCase() === k; });
            });
            hdef('get', function (k) { var x = this._key(k); return x === undefined ? null : this[x]; });
            hdef('has', function (k) { return this._key(k) !== undefined; });
            hdef('set', function (k, v) { var x = this._key(k); if (x !== undefined) delete this[x]; this[String(k)] = String(v); });
            hdef('append', function (k, v) { var x = this._key(k); if (x !== undefined) this[x] = this[x] + ', ' + v; else this[String(k)] = String(v); });
            hdef('delete', function (k) { var x = this._key(k); if (x !== undefined) delete this[x]; });
            hdef('forEach', function (cb, t) { var s = this; Object.keys(s).forEach(function (k) { cb.call(t, s[k], k.toLowerCase(), s); }); });
            hdef('entries', function () { var s = this; return Object.keys(s).map(function (k) { return [k.toLowerCase(), s[k]]; })[Symbol.iterator](); });
            hdef(Symbol.iterator, Headers.prototype.entries);

            function utf8Encode(s) {
                var out = [];
                for (var i = 0; i < s.length; i++) {
                    var c = s.codePointAt(i);
                    if (c > 0xffff) i++;
                    if (c < 0x80) out.push(c);
                    else if (c < 0x800) out.push(0xc0 | (c >> 6), 0x80 | (c & 63));
                    else if (c < 0x10000) out.push(0xe0 | (c >> 12), 0x80 | ((c >> 6) & 63), 0x80 | (c & 63));
                    else out.push(0xf0 | (c >> 18), 0x80 | ((c >> 12) & 63), 0x80 | ((c >> 6) & 63), 0x80 | (c & 63));
                }
                return new Uint8Array(out);
            }
            function utf8Decode(b) {
                b = b instanceof ArrayBuffer ? new Uint8Array(b) : (ArrayBuffer.isView(b) ? new Uint8Array(b.buffer, b.byteOffset, b.byteLength) : new Uint8Array(b || []));
                var s = '', i = 0;
                while (i < b.length) {
                    var c = b[i++], cp;
                    if (c < 0x80) cp = c;
                    else if (c < 0xe0) cp = ((c & 31) << 6) | (b[i++] & 63);
                    else if (c < 0xf0) cp = ((c & 15) << 12) | ((b[i++] & 63) << 6) | (b[i++] & 63);
                    else cp = ((c & 7) << 18) | ((b[i++] & 63) << 12) | ((b[i++] & 63) << 6) | (b[i++] & 63);
                    s += String.fromCodePoint(cp);
                }
                return s;
            }
            function TextEncoder() {}
            TextEncoder.prototype.encoding = 'utf-8';
            TextEncoder.prototype.encode = function (s) { return utf8Encode(String(s === undefined ? '' : s)); };
            function TextDecoder() {}
            TextDecoder.prototype.encoding = 'utf-8';
            TextDecoder.prototype.decode = function (b) { return utf8Decode(b); };

            // Timers queue up and are drained by the host (NovelPlugin.callMethod) only
            // while a call is still pending, so sleeps and timeout races both behave.
            var timers = [], timerId = 0;
            g.setTimeout = function (fn, ms) {
                var id = ++timerId;
                timers.push({ id: id, fn: fn, args: Array.prototype.slice.call(arguments, 2), due: Date.now() + (+ms || 0) });
                return id;
            };
            g.clearTimeout = function (id) { timers = timers.filter(function (t) { return t.id !== id; }); };
            g.__runNextTimer = function () {
                if (!timers.length) return false;
                timers.sort(function (a, b) { return a.due - b.due || a.id - b.id; });
                var t = timers.shift();
                var wait = t.due - Date.now();
                if (wait > 0) __hostSleep(wait);
                if (typeof t.fn === 'function') t.fn.apply(null, t.args);
                return true;
            };
            g.__clearTimers = function () { timers = []; };

            g.URLSearchParams = URLSearchParams;
            g.URL = URL;
            if (typeof g.Headers === 'undefined') g.Headers = Headers;
            if (typeof g.TextEncoder === 'undefined') g.TextEncoder = TextEncoder;
            if (typeof g.TextDecoder === 'undefined') g.TextDecoder = TextDecoder;

            var B64 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
            if (typeof g.atob === 'undefined') g.atob = function (s) {
                s = String(s).replace(/[\s=]/g, '').replace(/-/g, '+').replace(/_/g, '/');
                var out = '', bits = 0, acc = 0;
                for (var i = 0; i < s.length; i++) {
                    var v = B64.indexOf(s.charAt(i));
                    if (v < 0) throw new Error('atob: invalid base64');
                    acc = (acc << 6) | v; bits += 6;
                    if (bits >= 8) { bits -= 8; out += String.fromCharCode((acc >> bits) & 255); }
                }
                return out;
            };
            if (typeof g.btoa === 'undefined') g.btoa = function (s) {
                s = String(s);
                var out = '';
                for (var i = 0; i < s.length; i += 3) {
                    var a = s.charCodeAt(i), b = s.charCodeAt(i + 1), c = s.charCodeAt(i + 2);
                    if (a > 255 || b > 255 || c > 255) throw new Error('btoa: character out of range');
                    var n = (a << 16) | ((b || 0) << 8) | (c || 0);
                    out += B64.charAt(n >> 18) + B64.charAt((n >> 12) & 63) +
                        (i + 1 < s.length ? B64.charAt((n >> 6) & 63) : '=') +
                        (i + 2 < s.length ? B64.charAt(n & 63) : '=');
                }
                return out;
            };
        })(globalThis);
    """

    /** Backs setTimeout's delay; capped so a plugin can't stall a gRPC call indefinitely. */
    val hostSleep: ProxyExecutable = ProxyExecutable { args ->
        val ms = args.getOrNull(0)?.takeIf { it.isNumber }?.asDouble()?.toLong() ?: 0L
        if (ms > 0) Thread.sleep(ms.coerceAtMost(10_000))
        null
    }

    fun requireFn(storageNamespace: String, wrapPromise: Value): ProxyExecutable = ProxyExecutable { args ->
        when (args[0].asString()) {
            "cheerio" -> cheerioModule()
            "htmlparser2" -> htmlparser2Module()
            "@libs/fetch" -> fetchModule(wrapPromise)
            "@libs/novelStatus" -> novelStatusModule()
            "@libs/isAbsoluteUrl" -> isAbsoluteUrlModule()
            "@libs/defaultCover", "@/types/constants" -> defaultCoverModule()
            "@libs/utils" -> utilsModule()
            "@libs/filterInputs" -> filterInputsModule()
            "@libs/storage" -> storageModule(storageNamespace)
            "dayjs" -> dayjsModule()
            "@libs/aes" -> aesModule()
            else -> null
        }
    }

    private val DAYJS_FORMATS: List<java.time.format.DateTimeFormatter> = listOf(
        java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME,
        java.time.format.DateTimeFormatter.ISO_LOCAL_DATE_TIME,
        java.time.format.DateTimeFormatter.ISO_LOCAL_DATE,
        java.time.format.DateTimeFormatter.ofPattern("MMMM d, yyyy", java.util.Locale.ENGLISH),
        java.time.format.DateTimeFormatter.ofPattern("MMM d, yyyy", java.util.Locale.ENGLISH),
        java.time.format.DateTimeFormatter.ofPattern("d MMMM yyyy", java.util.Locale.ENGLISH),
        java.time.format.DateTimeFormatter.ofPattern("yyyy/MM/dd", java.util.Locale.ENGLISH),
        java.time.format.DateTimeFormatter.ofPattern("MM/dd/yyyy", java.util.Locale.ENGLISH),
        java.time.format.DateTimeFormatter.ofPattern("dd-MM-yyyy", java.util.Locale.ENGLISH),
    )

    private fun dayjsModule(): ProxyExecutable = ProxyExecutable { args ->
        dayjsObject(parseDayjsInput(args.getOrNull(0)))
    }

    private fun parseDayjsInput(v: Value?): java.time.LocalDateTime? {
        if (v == null || v.isNull) return java.time.LocalDateTime.now()
        if (v.isNumber) return java.time.LocalDateTime.ofInstant(
            java.time.Instant.ofEpochMilli(v.asLong()), java.time.ZoneOffset.UTC,
        )
        val s = (if (v.isString) v.asString() else v.toString()).trim()
        if (s.isEmpty()) return java.time.LocalDateTime.now()
        for (f in DAYJS_FORMATS) {
            val parsed = try {
                java.time.LocalDateTime.parse(s, f)
            } catch (_: Exception) {
                try { java.time.LocalDate.parse(s, f).atStartOfDay() } catch (_: Exception) { null }
            }
            if (parsed != null) return parsed
        }
        return null
    }

    private fun dayjsUnit(args: Array<Value>): String =
        (args.getOrNull(1)?.takeIf { !it.isNull }?.asString() ?: "millisecond")
            .lowercase().removeSuffix("s")

    private fun dayjsShift(dt: java.time.LocalDateTime, n: Long, unit: String): java.time.LocalDateTime =
        when (unit) {
            "year" -> dt.plusYears(n)
            "month" -> dt.plusMonths(n)
            "week" -> dt.plusWeeks(n)
            "day" -> dt.plusDays(n)
            "hour" -> dt.plusHours(n)
            "minute" -> dt.plusMinutes(n)
            "second" -> dt.plusSeconds(n)
            "millisecond" -> dt.plusNanos(n * 1_000_000)
            else -> dt
        }

    private fun dayjsFormat(dt: java.time.LocalDateTime, fmt: String): String = when (fmt) {
        "LL" -> dt.format(java.time.format.DateTimeFormatter.ofPattern("MMMM d, yyyy", java.util.Locale.ENGLISH))
        "LLL" -> dt.format(java.time.format.DateTimeFormatter.ofPattern("MMMM d, yyyy h:mm a", java.util.Locale.ENGLISH))
        "LLLL" -> dt.format(java.time.format.DateTimeFormatter.ofPattern("EEEE, MMMM d, yyyy h:mm a", java.util.Locale.ENGLISH))
        "L" -> dt.format(java.time.format.DateTimeFormatter.ofPattern("MM/dd/yyyy", java.util.Locale.ENGLISH))
        "LT" -> dt.format(java.time.format.DateTimeFormatter.ofPattern("h:mm a", java.util.Locale.ENGLISH))
        else -> {
            val javaPattern = fmt
                .replace("YYYY", "yyyy").replace("YY", "yy")
                .replace("DD", "dd").replace("D", "d")
                .replace("Z", "XXX")
            try {
                dt.format(java.time.format.DateTimeFormatter.ofPattern(javaPattern, java.util.Locale.ENGLISH))
            } catch (_: Exception) {
                dt.toString()
            }
        }
    }

    private fun dayjsObject(dt: java.time.LocalDateTime?): ProxyObject {
        val valid = dt != null
        val map = HashMap<String, Any>()
        map["isValid"] = ProxyExecutable { valid }
        map["valueOf"] = ProxyExecutable {
            if (valid) dt!!.toInstant(java.time.ZoneOffset.UTC).toEpochMilli() else Double.NaN
        }
        map["unix"] = ProxyExecutable {
            if (valid) dt!!.toEpochSecond(java.time.ZoneOffset.UTC) else Double.NaN
        }
        map["toISOString"] = ProxyExecutable {
            if (valid) dt!!.atOffset(java.time.ZoneOffset.UTC)
                .format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
            else "Invalid Date"
        }
        map["year"] = ProxyExecutable { if (valid) dt!!.year else Double.NaN }
        map["month"] = ProxyExecutable { if (valid) dt!!.monthValue - 1 else Double.NaN }
        map["date"] = ProxyExecutable { if (valid) dt!!.dayOfMonth else Double.NaN }
        map["format"] = ProxyExecutable { fargs ->
            if (!valid) return@ProxyExecutable "Invalid Date"
            val fmt = fargs.getOrNull(0)?.takeIf { !it.isNull }?.asString() ?: "yyyy-MM-dd'T'HH:mm:ssXXX"
            dayjsFormat(dt!!, fmt)
        }
        map["subtract"] = ProxyExecutable { sargs ->
            if (!valid) dayjsObject(null)
            else dayjsObject(dayjsShift(dt!!, -sargs[0].asLong(), dayjsUnit(sargs)))
        }
        map["add"] = ProxyExecutable { sargs ->
            if (!valid) dayjsObject(null)
            else dayjsObject(dayjsShift(dt!!, sargs[0].asLong(), dayjsUnit(sargs)))
        }
        return ProxyObject.fromMap(map)
    }

    private fun cheerioModule(): ProxyObject = ProxyObject.fromMap(mapOf(
        "load" to ProxyExecutable { args -> CheerioRoot.load(args) },
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

    // @libs/aes mirrors @noble/ciphers/aes (gcm, cbc, ctr, ecb with { encrypt, decrypt }); GCM appends the 16-byte tag.
    private fun aesModule(): ProxyObject {
        fun noPadding(opts: Value?) = opts != null && !opts.isNull && opts.hasMembers() && truthy(opts.getMember("disablePadding"))
        fun cipher(transform: String, key: Value, spec: java.security.spec.AlgorithmParameterSpec?, aad: Value? = null): ProxyObject {
            val keySpec = javax.crypto.spec.SecretKeySpec(bytesOf(key), "AES")
            fun run(mode: Int, data: Value): ByteArray {
                val c = javax.crypto.Cipher.getInstance(transform)
                if (spec == null) c.init(mode, keySpec) else c.init(mode, keySpec, spec)
                if (aad != null && !aad.isNull) c.updateAAD(bytesOf(aad))
                return c.doFinal(bytesOf(data))
            }
            return ProxyObject.fromMap(mapOf(
                "encrypt" to ProxyExecutable { a -> run(javax.crypto.Cipher.ENCRYPT_MODE, a[0]) },
                "decrypt" to ProxyExecutable { a -> run(javax.crypto.Cipher.DECRYPT_MODE, a[0]) },
            ))
        }
        return ProxyObject.fromMap(mapOf(
            "gcm" to ProxyExecutable { a ->
                cipher("AES/GCM/NoPadding", a[0], javax.crypto.spec.GCMParameterSpec(128, bytesOf(a[1])), a.getOrNull(2))
            },
            "cbc" to ProxyExecutable { a ->
                val padding = if (noPadding(a.getOrNull(2))) "NoPadding" else "PKCS5Padding"
                cipher("AES/CBC/$padding", a[0], javax.crypto.spec.IvParameterSpec(bytesOf(a[1])))
            },
            "ctr" to ProxyExecutable { a -> cipher("AES/CTR/NoPadding", a[0], javax.crypto.spec.IvParameterSpec(bytesOf(a[1]))) },
            "ecb" to ProxyExecutable { a ->
                val padding = if (noPadding(a.getOrNull(1))) "NoPadding" else "PKCS5Padding"
                cipher("AES/ECB/$padding", a[0], null)
            },
        ))
    }

    private fun bytesOf(v: Value): ByteArray = when {
        v.isString -> v.asString().toByteArray(Charsets.UTF_8)
        v.hasBufferElements() -> ByteArray(v.bufferSize.toInt()).also { v.readBuffer(0, it, 0, it.size) }
        v.hasArrayElements() -> ByteArray(v.arraySize.toInt()) { v.getArrayElement(it.toLong()).asInt().toByte() }
        else -> throw IllegalArgumentException("expected a Uint8Array, got $v")
    }

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
        // In LNReader these snapshot the site's WebView storage; there is no WebView here.
        "localStorage" to ProxyObject.fromMap(mapOf("get" to ProxyExecutable { null })),
        "sessionStorage" to ProxyObject.fromMap(mapOf("get" to ProxyExecutable { null })),
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
