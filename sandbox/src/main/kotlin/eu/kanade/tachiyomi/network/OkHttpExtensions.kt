package eu.kanade.tachiyomi.network

import okhttp3.Call
import okhttp3.Callback
import okhttp3.Response
import rx.Observable
import rx.Subscriber
import java.io.IOException

fun Call.asObservable(): Observable<Response> {
    return Observable.create { subscriber: Subscriber<in Response> ->
        val call = this.clone()
        call.enqueue(object : Callback {
            override fun onResponse(call: Call, response: Response) {
                subscriber.onNext(response)
                subscriber.onCompleted()
            }
            override fun onFailure(call: Call, e: IOException) {
                if (!subscriber.isUnsubscribed) subscriber.onError(e)
            }
        })
    }
}

fun Call.asObservableSuccess(): Observable<Response> {
    return asObservable().map { response ->
        if (!response.isSuccessful) {
            response.close()
            throw HttpException(response.code)
        }
        response
    }
}
