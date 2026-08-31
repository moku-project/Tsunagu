package android.os;

public final class Build {

    private Build() {}

    public static final String UNKNOWN = "unknown";

    public static final String BOARD = "sandbox";
    public static final String BOOTLOADER = UNKNOWN;
    public static final String BRAND = "google";
    public static final String DEVICE = "sandbox";
    public static final String DISPLAY = "TQ3A.230901.001";
    public static final String FINGERPRINT = "google/sandbox/sandbox:13/TQ3A.230901.001/1:user/release-keys";
    public static final String HARDWARE = "sandbox";
    public static final String HOST = "sandbox";
    public static final String ID = "TQ3A.230901.001";
    public static final String MANUFACTURER = "Google";
    public static final String MODEL = "Pixel 7";
    public static final String PRODUCT = "sandbox";
    public static final String TAGS = "release-keys";
    public static final String TYPE = "user";
    public static final String USER = "android-build";
    public static final long TIME = 1_693_526_400_000L;

    public static final String[] SUPPORTED_ABIS = { "arm64-v8a", "armeabi-v7a", "armeabi" };
    public static final String[] SUPPORTED_32_BIT_ABIS = { "armeabi-v7a", "armeabi" };
    public static final String[] SUPPORTED_64_BIT_ABIS = { "arm64-v8a" };

    public static final String SERIAL = UNKNOWN;

    public static String getRadioVersion() {
        return UNKNOWN;
    }

    public static final class VERSION {

        private VERSION() {}

        public static final String INCREMENTAL = "1";
        public static final String RELEASE = "13";
        public static final String RELEASE_OR_CODENAME = "13";
        public static final String RELEASE_OR_PREVIEW_DISPLAY = "13";
        public static final String BASE_OS = "";
        public static final String SECURITY_PATCH = "2023-09-01";
        public static final String SDK = "33";
        public static final int SDK_INT = 33;
        public static final int PREVIEW_SDK_INT = 0;
        public static final String CODENAME = "REL";
    }

    public static final class VERSION_CODES {

        private VERSION_CODES() {}

        public static final int BASE = 1;
        public static final int LOLLIPOP = 21;
        public static final int LOLLIPOP_MR1 = 22;
        public static final int M = 23;
        public static final int N = 24;
        public static final int N_MR1 = 25;
        public static final int O = 26;
        public static final int O_MR1 = 27;
        public static final int P = 28;
        public static final int Q = 29;
        public static final int R = 30;
        public static final int S = 31;
        public static final int S_V2 = 32;
        public static final int TIRAMISU = 33;
        public static final int UPSIDE_DOWN_CAKE = 34;
        public static final int CUR_DEVELOPMENT = 10000;
    }
}
