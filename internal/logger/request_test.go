package logger

import (
	"context"
	"testing"
)

// The id a request is correlated by travels in its context, and a context
// that never received one says so with an empty id rather than a made-up one.
func TestARequestIDTravelsInTheContext(t *testing.T) {
	t.Parallel()

	if got := RequestID(context.Background()); got != "" {
		t.Errorf("RequestID() of a context without one = %q, want empty", got)
	}
	ctx := WithRequestID(context.Background(), "01a0dd9a")
	if got := RequestID(ctx); got != "01a0dd9a" {
		t.Errorf("RequestID() = %q, want %q", got, "01a0dd9a")
	}
}
