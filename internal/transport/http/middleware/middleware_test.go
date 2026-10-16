package middleware_test

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/MathTrail/canvas-api/internal/transport/http/middleware"
)

// A probe that answers normally is not worth a line; a probe that fails is.
func TestSuccessfulProbesAreNotLogged(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	serve(t, router, http.MethodGet, "/health")

	if logs.Len() != 0 {
		t.Errorf("a successful probe wrote %d lines, want none", logs.Len())
	}
}

func TestFailuresAreLoggedWithTheirRequestID(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusServiceUnavailable) })

	serve(t, router, http.MethodGet, "/health")

	line := onlyLine(t, logs)
	if line.Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want %v for a 5xx", line.Level, zapcore.ErrorLevel)
	}
	if field(t, line, "request_id") == "" {
		t.Error("request_id is empty, want the id the request carried")
	}
}

// One request must not take the process down: another child is in the middle
// of a task on the same instance.
func TestAPanicBecomesAnAnswer(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/boom", func(_ *gin.Context) { panic("boom") })

	rec := serve(t, router, http.MethodGet, "/boom")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), "INTERNAL_ERROR") {
		t.Errorf("body = %s, want a structured error", rec.Body.String())
	}
	if rec.Body.String() != "" && strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("body = %s, want the panic's text kept out of the answer", rec.Body.String())
	}
	if line := panickedLine(t, logs); line.Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want %v for a panic", line.Level, zapcore.ErrorLevel)
	}
}

// Whatever a handler panicked with, the line says what kind of fault it was, as
// text in a field of a known type, and none of what the value carried: a value
// a handler panics with may hold a task or an answer. A fault of the runtime
// is told in the runtime's own words, which hold no data.
func TestAPanicIsLoggedAsText(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		panic func()
		want  string
	}{
		{name: "an error", panic: func() { panic(errors.New("the jug is empty")) },
			want: "a value of type *errors.errorString"},
		{name: "anything else", panic: func() { panic(42) }, want: "a value of type int"},
		{name: "a fault of the runtime", panic: func() { fill(nil, 1) },
			want: "runtime error: index out of range [1] with length 0"},
		{name: "a fault of the runtime, wrapped", panic: func() {
			panic(fmt.Errorf("while filling %q: %w", "the jug of the task", runtimeFault()))
		}, want: "runtime error: index out of range [1] with length 0"},
		{name: "a value that calls itself a fault of the runtime", panic: func() { panic(impostor{}) },
			want: "a value of type middleware_test.impostor"},
		{name: "a fault of the runtime behind a pointer", panic: func() { countOf("six") },
			want: "interface conversion: interface {} is string, not int"},
		{name: "a value behind a pointer that calls itself a fault of the runtime", panic: func() { panic(&impostor{}) },
			want: "a value of type *middleware_test.impostor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logs, router := routerWithObservedLogs(t)
			router.GET("/boom", func(_ *gin.Context) { tc.panic() })

			serve(t, router, http.MethodGet, "/boom")

			// field fails the test unless the value is a string, which is the
			// half of this that matters: a log field has a type on purpose.
			if got := field(t, panickedLine(t, logs), "panic"); got != tc.want {
				t.Errorf("panic = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func routerWithObservedLogs(t *testing.T) (*observer.ObservedLogs, *gin.Engine) {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	router := gin.New()
	routes := middleware.NewRoutes(router)
	router.Use(middleware.LastResort(logger, routes))
	router.Use(middleware.RequestID())
	router.Use(middleware.ZapLogger(logger, routes))
	router.Use(middleware.ZapRecovery())
	return logs, router
}

// panickedLine is the one line a panicked request is logged in: its request
// line, carrying what it panicked with, and no second line beside it.
func panickedLine(t *testing.T, logs *observer.ObservedLogs) *observer.LoggedEntry {
	t.Helper()

	if others := logs.FilterMessage("panic").Len(); others != 0 {
		t.Errorf("panic lines = %d beside the request line, want none", others)
	}
	line := onlyLine(t, logs)
	if _, found := line.ContextMap()["panic"]; !found {
		t.Fatalf("the request line says nothing of the panic: %v", line.ContextMap())
	}
	return line
}

// fill writes into a list of jugs, and indexes past the end of one too short:
// a fault of the runtime that no analysis of the code can see coming.
func fill(jugs []int, at int) { jugs[at] = 1 }

// keep puts a value in a set by the value itself, which the runtime refuses from
// inside its own map code when the value cannot be hashed.
func keep(set map[any]bool, value any) { set[value] = true }

// countOf reads an answer as a count, which the runtime refuses when it is not
// one — with the one fault it raises behind a pointer.
func countOf(answer any) int {
	return answer.(int) //nolint:errcheck // the refusal is the fault under test
}

// runtimeFault is the fault fill makes of a list too short, caught as a value.
func runtimeFault() error {
	var fault error
	func() {
		defer func() {
			if recovered, isError := recover().(error); isError {
				fault = recovered
			}
		}()
		fill(nil, 1)
	}()
	return fault
}

// impostor is a value that calls itself a fault of the runtime, and is not one.
type impostor struct{}

func (impostor) Error() string { return "the answer is 36" }
func (impostor) RuntimeError() {}

func serve(t *testing.T, router *gin.Engine, method, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, target, http.NoBody)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func onlyLine(t *testing.T, logs *observer.ObservedLogs) *observer.LoggedEntry {
	t.Helper()

	entries := logs.FilterMessage("http_request").All()
	if len(entries) != 1 {
		t.Fatalf("http_request lines = %d, want exactly 1", len(entries))
	}
	return &entries[0]
}

func field(t *testing.T, line *observer.LoggedEntry, key string) string {
	t.Helper()

	value, found := line.ContextMap()[key]
	if !found {
		t.Fatalf("the line has no %q field: %v", key, line.ContextMap())
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%q = %T, want a string", key, value)
	}
	return text
}

// The errors a request collected are written by what they were, not by what
// they said: a write that failed names both ends of the connection, and an
// error of the service's own can carry whatever its code had in hand.
func TestACollectedErrorIsLoggedByItsKind(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/written", func(c *gin.Context) {
		_ = c.Error(&net.OpError{
			Op:     "write",
			Net:    "tcp",
			Source: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 2), Port: 8080},
			Addr:   &net.TCPAddr{IP: net.IPv4(203, 0, 113, 9), Port: 51234},
			Err:    os.NewSyscallError("write", syscall.EPIPE),
		})
		_ = c.Error(errors.New("gemini: read the sheet of Otter"))
		c.Status(http.StatusOK)
	})

	serve(t, router, http.MethodGet, "/written")

	line := onlyLine(t, logs).ContextMap()
	want := []any{"broken pipe", "an error of type *errors.errorString"}
	if got, isList := line["errors"].([]any); !isList || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("errors = %v, want %v", line["errors"], want)
	}
	for _, kept := range []string{"203.0.113.9", "Otter"} {
		if strings.Contains(fmt.Sprint(line), kept) {
			t.Errorf("the line carries %q, want the errors by their kind alone: %v", kept, line)
		}
	}
}

// A panic in the middleware itself, above the recovery next to the handlers,
// is caught by the last resort: logged once, by its kind alone, and answered.
func TestAPanicAboveTheRecoveryIsCaughtByTheLastResort(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.DebugLevel)
	router := gin.New()
	router.Use(middleware.LastResort(zap.New(core), middleware.NewRoutes(router)))
	router.Use(middleware.RequestID())
	router.Use(func(c *gin.Context) {
		c.Next()
		panic(errors.New("the answer is 36"))
	})
	router.Use(middleware.ZapRecovery())
	router.GET("/fine", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	serve(t, router, http.MethodGet, "/fine")

	entries := logs.FilterMessage("panic").All()
	if len(entries) != 1 {
		t.Fatalf("panic lines = %d, want exactly 1", len(entries))
	}
	if got := field(t, &entries[0], "panic"); got != "a value of type *errors.errorString" {
		t.Errorf("panic = %q, want its type alone", got)
	}
	if field(t, &entries[0], "request_id") == "" {
		t.Error("request_id is empty, want the id the request carried")
	}
}

// Both lines that name a request — its own line, and the last resort's when
// the middleware above the recovery panicked — name it by the route it matched
// and by a method from the known ones, never by what the caller wrote: a path
// carries identifiers, a path nothing matched is text the caller chose, and a
// verb can be any word at all.
func TestALineNamesTheRouteAndNotThePath(t *testing.T) {
	t.Parallel()

	for _, kind := range []struct {
		name string
		line func(t *testing.T, method, target string) *observer.LoggedEntry
	}{
		{name: "the request's own line", line: requestLine},
		{name: "the last resort's line", line: lastResortLine},
	} {
		for _, tc := range []struct {
			name, method, target, route, verb string
		}{
			{"a route with a name in it", http.MethodGet, "/tasks/masha-ivanova", "/tasks/:id", http.MethodGet},
			{"a path nothing matched", http.MethodPost,
				(&url.URL{Path: "/x\n{\"severity\":\"ERROR\",\"message\":\"masha ivanova\"}"}).EscapedPath(),
				"other", http.MethodPost},
			{"a verb nobody defined", "MASHA", "/tasks/42", "other", "other"},
		} {
			t.Run(kind.name+", "+tc.name, func(t *testing.T) {
				t.Parallel()

				line := kind.line(t, tc.method, tc.target)
				if got := field(t, line, "route"); got != tc.route {
					t.Errorf("route = %q, want %q", got, tc.route)
				}
				if got := field(t, line, "method"); got != tc.verb {
					t.Errorf("method = %q, want %q", got, tc.verb)
				}
				if written := strings.ToLower(fmt.Sprint(line.ContextMap())); strings.Contains(written, "masha") {
					t.Errorf("the line carries what the caller wrote: %v", line.ContextMap())
				}
			})
		}
	}
}

// requestLine serves one request through the request log and gives back its
// line.
func requestLine(t *testing.T, method, target string) *observer.LoggedEntry {
	t.Helper()

	logs, router := routerWithObservedLogs(t)
	router.GET("/tasks/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	serve(t, router, method, target)
	return onlyLine(t, logs)
}

// lastResortLine serves one request through middleware that panics above the
// recovery, and gives back the line the last resort wrote.
func lastResortLine(t *testing.T, method, target string) *observer.LoggedEntry {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)
	router := gin.New()
	router.Use(middleware.LastResort(zap.New(core), middleware.NewRoutes(router)))
	router.Use(func(*gin.Context) { panic("the middleware is on fire") })
	router.GET("/tasks/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	serve(t, router, method, target)

	entries := logs.FilterMessage("panic").All()
	if len(entries) != 1 {
		t.Fatalf("panic lines = %d, want exactly 1", len(entries))
	}
	return &entries[0]
}

// An answer with no body has a body of zero, not the -1 the writer starts at.
func TestAnEmptyBodyIsLoggedAsZero(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/gone", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	serve(t, router, http.MethodGet, "/gone")

	size, found := onlyLine(t, logs).ContextMap()["body_size"]
	if !found {
		t.Fatal("the line has no body_size field")
	}
	if size != int64(0) {
		t.Errorf("body_size = %v, want 0", size)
	}
}

// A handler that panicked halfway through its answer has already sent a status
// and headers, and a second one would corrupt what the client is reading.
func TestAPanicAfterTheAnswerHasStartedDoesNotWriteTwice(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/half", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"first": "part"})
		panic("boom")
	})

	rec := serve(t, router, http.MethodGet, "/half")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want the %d the handler already sent", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "INTERNAL_ERROR") {
		t.Errorf("body = %s, want no second answer appended to the first", rec.Body.String())
	}
	if line := panickedLine(t, logs); line.Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want %v for a panic, whatever the answer had got to", line.Level, zapcore.ErrorLevel)
	}
}

// A probe that answers is not worth a line, and a probe that panicked is.
func TestAPanickedProbeIsLogged(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
		c.Writer.WriteHeaderNow()
		panic("boom")
	})

	serve(t, router, http.MethodGet, "/health")

	if line := panickedLine(t, logs); line.Level != zapcore.ErrorLevel {
		t.Errorf("level = %v, want %v", line.Level, zapcore.ErrorLevel)
	}
}

// net/http documents ErrAbortHandler as the way to drop a request silently.
// It is passed on to the server bare, however it was wrapped, past everything
// that counts and logs: a request that asked for no answer is neither a panic
// nor an answer of 200.
func TestASilentAbortIsPassedOnToTheServer(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		abort error
	}{
		{"an abort", http.ErrAbortHandler},
		{"an abort wrapped on the way", fmt.Errorf("proxy: %w", http.ErrAbortHandler)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logs, router := routerWithObservedLogs(t)
			router.GET("/abort", func(_ *gin.Context) { panic(tc.abort) })

			if passed := servedPanic(t, router, "/abort"); passed != http.ErrAbortHandler { //nolint:errorlint // the server compares it by identity
				t.Errorf("the server was handed %v, want %v", passed, http.ErrAbortHandler)
			}
			if lines := logs.Len(); lines != 0 {
				t.Errorf("log lines = %d: %v, want none for a request that asked for no answer", lines, logs.All())
			}
		})
	}
}

// servedPanic serves one request, and returns what panicked out of the router
// to the server, if anything did.
func servedPanic(t *testing.T, router *gin.Engine, target string) (passed any) {
	t.Helper()

	defer func() { passed = recover() }()
	serve(t, router, http.MethodGet, target)
	return nil
}

// A connection that died under the service's own call — to Gemini, say — is
// a fault of the request, not the client hanging up: the client is still
// there, and it is answered and the fault written down.
func TestAConnectionOfOurOwnGoneIsAFault(t *testing.T) {
	t.Parallel()

	logs, router := routerWithObservedLogs(t)
	router.GET("/check", func(_ *gin.Context) {
		panic(&net.OpError{Op: "read", Err: os.NewSyscallError("read", syscall.ECONNRESET)})
	})

	if rec := serve(t, router, http.MethodGet, "/check"); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	panickedLine(t, logs)
}

// An id a client sends is never kept, however well it is formed: the answer
// and the request's line in the log carry an id of the service's own, so that
// nothing a caller wrote reaches a log line by way of its id.
func TestAClientsIDIsNeverKept(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		id   string
	}{
		{"a well-formed id", "01a0c467-8008-75c3-825b-37f804bf11bb"},
		{"a line break", "abc\r\nX-Injected: yes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logs, router := routerWithObservedLogs(t)
			router.GET("/x", func(c *gin.Context) { c.Status(http.StatusBadRequest) })
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", http.NoBody)
			req.Header[middleware.RequestIDHeader] = []string{tc.id}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			echoed := rec.Header().Get(middleware.RequestIDHeader)
			if echoed == tc.id || !serviceID(echoed) {
				t.Errorf("X-Request-Id = %q, want an id of the service's own, a UUID v7, not %q", echoed, tc.id)
			}
			if logged := field(t, onlyLine(t, logs), "request_id"); logged != echoed {
				t.Errorf("request_id = %q, want the id the answer carries, %q", logged, echoed)
			}
		})
	}
}

// A request the id middleware never saw has no id, and neither has a context
// with no request at all.
func TestNoIDIsReadWhereNoneWasSet(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		request *http.Request
	}{
		{name: "no request", request: nil},
		{name: "a request the middleware never saw",
			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", http.NoBody)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = tc.request
			if id := middleware.RequestIDFrom(c); id != "" {
				t.Errorf("RequestIDFrom() = %q, want no id", id)
			}
		})
	}
}

// serviceID says whether id is one the service hands out while it has
// randomness: a version 7 UUID.
func serviceID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.Version() == 7 && parsed.String() == id
}

// panickingFunction is named so that the stack the recovery logs can be held
// to starting at the code that panicked rather than at the machinery that
// caught it.
func panickingFunction() { panic("boom") }

// The stack starts at the code that panicked, whether it panicked by name or
// the runtime raised the fault, with neither the recovery nor the runtime's
// own frames above it: the first line a reader sees is the line that failed.
func TestThePanicStackStartsAtTheCodeThatPanicked(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, want string
		panic      func()
	}{
		{"a panic by name", "panickingFunction", panickingFunction},
		{"a fault of the runtime", "middleware_test.fill", func() { fill(nil, 1) }},
		{"a fault inside the runtime's own packages", "middleware_test.keep", func() { keep(map[any]bool{}, []int{1}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logs, router := routerWithObservedLogs(t)
			router.GET("/boom", func(_ *gin.Context) { tc.panic() })

			serve(t, router, http.MethodGet, "/boom")

			stack := field(t, panickedLine(t, logs), "stack")
			first, _, _ := strings.Cut(stack, "\n")
			if !strings.Contains(first, tc.want) {
				t.Errorf("the stack starts at %q, want %s; whole stack:\n%s", first, tc.want, stack)
			}
		})
	}
}

// Every answer carries what tells a browser how to treat it, a panic's answer
// included: the headers are set before anything below them runs.
func TestEveryAnswerCarriesTheSecurityHeaders(t *testing.T) {
	t.Parallel()

	_, router := routerWithObservedLogs(t)
	router.Use(middleware.SecurityHeaders())
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/panics", func(*gin.Context) { panic("boom") })

	for _, target := range []string{"/ok", "/panics"} {
		rec := serve(t, router, http.MethodGet, target)
		for header, want := range map[string]string{
			"Strict-Transport-Security": "max-age=31536000",
			"X-Content-Type-Options":    "nosniff",
			"Referrer-Policy":           "no-referrer",
		} {
			if got := rec.Header().Get(header); got != want {
				t.Errorf("GET %s: %s = %q, want %q", target, header, got, want)
			}
		}
	}
}

// A handler that panics after it set headers for the answer it meant to give
// gets the error answered as JSON all the same: its type and its length are
// dropped, and what the chain put on every answer stays.
func TestAPanicDropsTheHeadersTheHandlerSet(t *testing.T) {
	t.Parallel()

	_, router := routerWithObservedLogs(t)
	router.GET("/image", func(c *gin.Context) {
		c.Header("Content-Type", "image/png")
		c.Header("Content-Length", "1234")
		c.Header("Cache-Control", "public, max-age=31536000")
		panic("boom")
	})

	rec := serve(t, router, http.MethodGet, "/image")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type = %q, want the error's own application/json", got)
	}
	for _, dropped := range []string{"Content-Length", "Cache-Control"} {
		if got := rec.Header().Get(dropped); got != "" {
			t.Errorf("%s = %q, want the handler's value dropped", dropped, got)
		}
	}
	if rec.Header().Get(middleware.RequestIDHeader) == "" {
		t.Errorf("%s is empty, want the request's id kept", middleware.RequestIDHeader)
	}
}
