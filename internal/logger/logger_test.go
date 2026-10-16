package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// The JSON encoder exists to be read by Cloud Logging without any further
// configuration, and that means three key names and one severity scale. A test
// that only checked "it logs something" would not notice the day one of them
// is renamed back to zap's default.
func TestJSONEncoderUsesCloudRunKeys(t *testing.T) {
	t.Parallel()

	encoder := zapcore.NewJSONEncoder(cloudRunEncoderConfig())
	entry := zapcore.Entry{
		Level:   zapcore.WarnLevel,
		Time:    time.Date(2026, 9, 21, 10, 30, 0, 0, time.UTC),
		Message: "http_request",
	}

	buf, err := encoder.EncodeEntry(entry, []zapcore.Field{{
		Key:    "route",
		Type:   zapcore.StringType,
		String: "/health",
	}})
	if err != nil {
		t.Fatalf("EncodeEntry() error = %v, want nil", err)
	}

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("the line is not JSON: %v; line = %s", err, buf.String())
	}

	for key, want := range map[string]string{
		"severity": "WARNING",
		"message":  "http_request",
		"time":     "2026-09-21T10:30:00Z",
		"route":    "/health",
	} {
		got, ok := line[key].(string)
		if !ok {
			t.Errorf("line[%q] is missing; line = %s", key, buf.String())
			continue
		}
		if got != want {
			t.Errorf("line[%q] = %q, want %q", key, got, want)
		}
	}

	for _, absent := range []string{"level", "msg", "ts"} {
		if _, found := line[absent]; found {
			t.Errorf("line still has zap's default key %q; line = %s", absent, buf.String())
		}
	}
}

func TestSeverityNames(t *testing.T) {
	t.Parallel()

	cases := map[zapcore.Level]string{
		zapcore.DebugLevel:  "DEBUG",
		zapcore.InfoLevel:   "INFO",
		zapcore.WarnLevel:   "WARNING",
		zapcore.ErrorLevel:  "ERROR",
		zapcore.DPanicLevel: "CRITICAL",
		zapcore.PanicLevel:  "CRITICAL",
		zapcore.FatalLevel:  "EMERGENCY",
		// A level the scale has no name for is written as the collector's own
		// word for "not said", rather than as nothing.
		zapcore.Level(42): "DEFAULT",
	}

	for level, want := range cases {
		t.Run(level.String(), func(t *testing.T) {
			t.Parallel()
			encoder := zapcore.NewJSONEncoder(cloudRunEncoderConfig())
			buf, err := encoder.EncodeEntry(zapcore.Entry{Level: level, Message: "x"}, nil)
			if err != nil {
				t.Fatalf("EncodeEntry() error = %v, want nil", err)
			}
			var line map[string]any
			if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
				t.Fatalf("the line is not JSON: %v", err)
			}
			if got := line["severity"]; got != want {
				t.Errorf("severity = %v, want %q", got, want)
			}
		})
	}
}

// A level or a format the logger does not know is refused rather than guessed
// at: a service whose log goes somewhere other than it was asked to is one
// nobody can see.
func TestNewRefusesWhatItDoesNotKnow(t *testing.T) {
	t.Parallel()

	for _, c := range []struct{ level, format string }{
		{level: "loud", format: "json"},
		{level: "info", format: "yaml"},
		{level: "info", format: ""},
	} {
		if log, err := New(c.level, c.format); err == nil {
			t.Errorf("New(%q, %q) = %v, want a refusal", c.level, c.format, log)
		}
	}
}

// The level asked for is the level written: info and nothing below it.
func TestNewWritesTheLevelAskedFor(t *testing.T) {
	t.Parallel()

	log, err := New("info", "json")
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if !log.Core().Enabled(zapcore.InfoLevel) {
		t.Error("info is disabled, want it enabled")
	}
	if log.Core().Enabled(zapcore.DebugLevel) {
		t.Error("debug is enabled, want nothing below info")
	}
}

// Several settings of the configuration are decisions, not defaults, and all are
// easy to lose to a future refactor that starts from zap's own constructors.
func TestConfigurationDecisions(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"json", "console"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			cfg, err := newConfig("info", format)
			if err != nil {
				t.Fatalf("newConfig() error = %v, want nil", err)
			}

			for _, decision := range []struct {
				name string
				got  any
				want any
			}{
				// The counts are taken from these lines.
				{name: "sampling off", got: cfg.Sampling == nil, want: true},
				// The only stacks are the ones the service writes.
				{name: "zap's own stacks switched off", got: cfg.DisableStacktrace, want: true},
				// The format changes nothing but the format.
				{name: "development mode", got: cfg.Development, want: false},
				{name: "output", got: fmt.Sprint(cfg.OutputPaths), want: "[stdout]"},
				{name: "the logger's own failures", got: fmt.Sprint(cfg.ErrorOutputPaths), want: "[stderr]"},
			} {
				if decision.got != decision.want {
					t.Errorf("%s = %v, want %v", decision.name, decision.got, decision.want)
				}
			}
		})
	}
}

// A field can never end the line it is in, whatever it carries. The code that
// writes each field keeps a caller's words out of it as far as it knows how,
// and the encoder is what keeps anything that still gets through harmless.
//
// The attack is a caller ending the line they are in and starting one of their
// own — a path of "/x\n{severity: ERROR, …}" becoming a second entry that
// nobody wrote. A JSON encoder escapes the newline, so the forgery stays inside
// the string it arrived in, and the fields it tried to forge keep the values
// this service gave them.
//
// This is the test that holds the encoder to that: swap it for one that does
// not escape and it goes red.
func TestAFieldCannotForgeASecondLine(t *testing.T) {
	t.Parallel()

	var written bytes.Buffer
	log := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(cloudRunEncoderConfig()),
		zapcore.AddSync(&written),
		zapcore.DebugLevel,
	))

	forged := "/x\n{\"severity\":\"ERROR\",\"message\":\"a line nobody wrote\"}"
	log.Info("http_request", zap.String("path", forged))

	if extra := strings.Count(strings.TrimRight(written.String(), "\n"), "\n"); extra != 0 {
		t.Errorf("the log holds %d lines, want exactly one", extra+1)
	}

	var entry map[string]any
	if err := json.Unmarshal(written.Bytes(), &entry); err != nil {
		t.Fatalf("what was written is not one JSON object: %v", err)
	}
	if entry["path"] != forged {
		t.Errorf("path = %q, want it kept whole and escaped", entry["path"])
	}
	if entry["message"] != "http_request" {
		t.Errorf("message = %q, want the forgery unable to replace it", entry["message"])
	}
	if entry["severity"] != "INFO" {
		t.Errorf("severity = %q, want the forgery unable to raise it", entry["severity"])
	}
}
