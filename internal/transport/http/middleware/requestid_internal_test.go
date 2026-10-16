package middleware

import (
	"errors"
	"regexp"
	"testing"
	"testing/iotest"

	"github.com/google/uuid"
)

// When the source of randomness stops answering there is still a request to
// correlate. The identifier that follows is the clock and a counter, in base
// 36, and two of them have to be different.
func TestTheFallbackIdentifierIsUsableAndUnique(t *testing.T) {
	t.Parallel()

	first := fallbackRequestID()
	second := fallbackRequestID()

	for _, id := range []string{first, second} {
		if !fallbackShape.MatchString(id) {
			t.Errorf("fallbackRequestID() = %q, want the clock and a counter in base 36", id)
		}
	}
	if first == second {
		t.Errorf("two identifiers came back the same: %q", first)
	}
}

// A request that arrives once the source of randomness has stopped answering
// is still given an identifier: the fallback, just as different from the one
// before it.
func TestAnIdentifierIsMadeWithoutRandomness(t *testing.T) {
	// Not parallel: the source of randomness belongs to the whole process, and
	// it is put back before any test that runs in parallel starts.
	uuid.SetRand(iotest.ErrReader(errors.New("no randomness left")))
	t.Cleanup(func() { uuid.SetRand(nil) })

	first, second := newRequestID(), newRequestID()
	for _, id := range []string{first, second} {
		if !fallbackShape.MatchString(id) {
			t.Errorf("newRequestID() = %q, want the fallback, the clock and a counter in base 36", id)
		}
	}
	if first == second {
		t.Errorf("two requests were given the same id %q, want them told apart", first)
	}
}

// fallbackShape is the id made without randomness: the clock and a counter,
// each in lower-case base 36.
var fallbackShape = regexp.MustCompile(`^[0-9a-z]+-[0-9a-z]+$`)
