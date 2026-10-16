package logger

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// impostor calls itself a fault of the runtime, and is not one.
type impostor struct{}

func (impostor) Error() string { return "the answer is 36" }
func (impostor) RuntimeError() {}

// outOfRange is a fault the runtime raises itself, caught as a value.
func outOfRange() (fault error) {
	defer func() {
		fault, _ = recover().(error)
	}()
	fill(nil, 1)
	return nil
}

// fill writes into a list of jugs, past its end when the list is too short.
func fill(jugs []int, at int) { jugs[at] = 1 }

// A panic is named in words that hold no data: the runtime's own for a fault it
// raised, wherever that fault was wrapped, and the type alone for anything
// else — which may hold a task, an answer or an image.
func TestAPanicIsNamedByWhatItIs(t *testing.T) {
	t.Parallel()

	fault := outOfRange()
	cases := []struct {
		name      string
		recovered any
		want      string
	}{
		{name: "a fault of the runtime", recovered: fault, want: fault.Error()},
		{name: "a fault of the runtime, wrapped", recovered: fmt.Errorf("filling the jug of masha: %w", fault), want: fault.Error()},
		{name: "an error of ours", recovered: errors.New("the answer is C"), want: "a value of type *errors.errorString"},
		{name: "a text", recovered: "the answer is C", want: "a value of type string"},
		{name: "a value that calls itself a fault of the runtime", recovered: impostor{}, want: "a value of type logger.impostor"},
		{name: "the same behind a pointer", recovered: &impostor{}, want: "a value of type *logger.impostor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := PanicValue(tc.recovered).String; got != tc.want {
				t.Errorf("panic = %q, want %q", got, tc.want)
			}
		})
	}
}

// The stack of a panic begins at the code that panicked, not at the recovery
// that caught it or in the runtime between the two.
func TestAPanicsStackBeginsWhereItPanicked(t *testing.T) {
	t.Parallel()

	var stack zap.Field
	func() {
		defer func() {
			if recover() != nil {
				stack = PanicStack()
			}
		}()
		panickingCode()
	}()

	first, _, _ := strings.Cut(stack.String, "\n")
	if !strings.HasSuffix(first, ".panickingCode") {
		t.Errorf("the stack begins at %q, want the code that panicked", first)
	}
}

// panickingCode is the code a stack is expected to begin at.
func panickingCode() { panic("the answer is C") }
