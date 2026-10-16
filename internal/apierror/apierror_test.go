package apierror_test

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/MathTrail/canvas-api/internal/apierror"
)

// The body is flat: a code and a message, and nothing else when there are no
// details, so that a client reading an error never meets a field it was not
// told about.
func TestAResponseWithoutDetailsHasTheCodeAndTheMessageOnly(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(apierror.Response{Code: apierror.CodeNotFound, Message: "no such endpoint"})
	if err != nil {
		t.Fatalf("Marshal() error = %v, want nil", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("Unmarshal() error = %v, want nil", err)
	}
	want := map[string]any{"code": apierror.CodeNotFound, "message": "no such endpoint"}
	if !maps.Equal(fields, want) {
		t.Errorf("body = %s, want exactly the code and the message", data)
	}
}

// Details travel as an object beside the code and the message, keeping the
// type of every value.
func TestDetailsTravelBesideTheCode(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(apierror.Response{
		Code:    apierror.CodeNotFound,
		Message: "no such endpoint",
		Details: map[string]any{"expected": 3, "field": "task_id"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v, want nil", err)
	}

	var body struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("Unmarshal() error = %v, want nil", err)
	}
	if body.Code != apierror.CodeNotFound {
		t.Errorf("code = %q, want %q", body.Code, apierror.CodeNotFound)
	}
	want := map[string]any{"expected": float64(3), "field": "task_id"}
	if !maps.Equal(body.Details, want) {
		t.Errorf("details = %v, want %v; body = %s", body.Details, want, data)
	}
}

// A code is what a client branches on, so what it says on the wire is part of
// the contract: a code renamed in the source would break every client that
// knew the old one while the tests that compare with the constant stayed green.
func TestTheCodesSayWhatTheySaid(t *testing.T) {
	t.Parallel()

	for code, want := range map[string]string{
		apierror.CodeInternal:         "INTERNAL_ERROR",
		apierror.CodeNotFound:         "NOT_FOUND",
		apierror.CodeMethodNotAllowed: "METHOD_NOT_ALLOWED",
	} {
		t.Run(want, func(t *testing.T) {
			t.Parallel()
			if code != want {
				t.Errorf("code = %q, want %q", code, want)
			}
		})
	}
}
