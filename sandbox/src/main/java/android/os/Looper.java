package android.os;

public final class Looper {

    private static final Looper MAIN = new Looper();

    private Looper() {}

    public static void prepare() {}

    public static void prepareMainLooper() {}

    public static Looper getMainLooper() {
        return MAIN;
    }

    public static Looper myLooper() {
        return null;
    }

    public static void loop() {}

    public static boolean loopOnce() {
        return false;
    }

    public Thread getThread() {
        return Thread.currentThread();
    }

    public void quit() {}

    public void quitSafely() {}

    @Override
    public String toString() {
        return "Looper (sandbox stub)";
    }
}
