package logger

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// maxStackDepth is how many frames of a panicking goroutine's stack are read.
const maxStackDepth = 64

// PanicValue names what a panic was raised with, and none of what the code
// that raised it carried. A fault of the runtime is put in the runtime's own
// words, found inside whatever wrapped it, because those words hold types and
// numbers and no data. Anything else is a value of ours or a library's, which
// may hold a task, an answer or an image, so only its type is written: the
// stack beside it shows where it came from.
func PanicValue(recovered any) zap.Field {
	var fault runtime.Error
	if err, isError := recovered.(error); isError && errors.As(err, &fault) && ofTheRuntime(fault) {
		return zap.String("panic", fault.Error())
	}
	return zap.String("panic", fmt.Sprintf("a value of type %T", recovered))
}

// ofTheRuntime says whether a fault was made by the Go runtime itself. Any type
// can call itself a runtime error by having the method, and one that is not
// the runtime's may say anything at all.
func ofTheRuntime(fault runtime.Error) bool {
	kind := reflect.TypeOf(fault)
	for kind.Kind() == reflect.Pointer {
		kind = kind.Elem()
	}
	return kind.PkgPath() == "runtime"
}

// PanicStack is the stack of the goroutine that is panicking, from the code
// that panicked down. It is read from inside the deferred function that
// recovered, while the panic is still on the stack. What stands above the
// panicking code is left off: the recovery that caught the panic, and the
// runtime's own frames — one for a panic called by name, a few more for a
// fault the runtime raised itself — so that the first line a reader sees is
// the line that failed.
func PanicStack() zap.Field {
	callers := make([]uintptr, maxStackDepth)
	frames := runtime.CallersFrames(callers[:runtime.Callers(1, callers)])

	var stack strings.Builder
	panicking, reached := false, false
	for {
		frame, more := frames.Next()
		switch {
		case reached:
		case frame.Function == "runtime.gopanic":
			panicking = true
		case panicking && !strings.HasPrefix(frame.Function, "runtime."):
			reached = true
		}
		if reached {
			fmt.Fprintf(&stack, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		if !more {
			return zap.String("stack", stack.String())
		}
	}
}
