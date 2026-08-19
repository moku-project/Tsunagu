package tsunagu.util

import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.serialization.json.JsonObject
import rx.Observable
import rx.Subscription
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

suspend fun <T> Observable<T>.awaitSingle(): T =
    suspendCancellableCoroutine { cont ->
        var sub: Subscription? = null
        sub = subscribe(
            { value -> cont.resume(value) },
            { err -> cont.resumeWithException(err) },
        )
        cont.invokeOnCancellation { sub?.unsubscribe() }
    }

val EmptyJsonObject = JsonObject(emptyMap())
