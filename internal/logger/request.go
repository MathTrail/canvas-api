package logger

import "context"

// requestIDKey holds the id of the request a context belongs to. It is a type
// of its own, so that nothing else a context carries can be taken for it.
type requestIDKey struct{}

// WithRequestID returns a copy of ctx that carries the id every line of one
// request shares. The id travels in the context rather than beside it because
// a line is often written far from the code that received the request — inside
// a handler another library calls — and the context is the one thing that
// reaches that far.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the id WithRequestID put in ctx, or an empty string.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
