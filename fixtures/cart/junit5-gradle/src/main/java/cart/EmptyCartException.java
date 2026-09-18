package cart;

public final class EmptyCartException extends RuntimeException {

    public EmptyCartException(String message) {
        super(message);
    }
}
