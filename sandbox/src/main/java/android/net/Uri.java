package android.net;

public class Uri {
    private final String uriString;

    private Uri(String uriString) {
        this.uriString = uriString;
    }

    public static Uri parse(String uriString) {
        return new Uri(uriString);
    }

    @Override
    public String toString() {
        return uriString;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (!(o instanceof Uri)) return false;
        return uriString.equals(((Uri) o).uriString);
    }

    @Override
    public int hashCode() {
        return uriString.hashCode();
    }
}
