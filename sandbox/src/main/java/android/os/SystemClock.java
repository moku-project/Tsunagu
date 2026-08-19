package android.os;

public final class SystemClock {

    private SystemClock() {}

    public static long elapsedRealtime() {
        return System.nanoTime() / 1_000_000L;
    }

    public static long elapsedRealtimeNanos() {
        return System.nanoTime();
    }

    public static long uptimeMillis() {
        return System.nanoTime() / 1_000_000L;
    }

    public static long currentThreadTimeMillis() {
        return System.nanoTime() / 1_000_000L;
    }

    public static boolean sleep(long ms) {
        try {
            Thread.sleep(ms);
            return true;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return false;
        }
    }
}
