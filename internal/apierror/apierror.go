// Package apierror defines the shape of an HTTP error response.
//
// One shape for every error the service answers with: a stable
// machine-readable code, a sentence meant to be relayed to a person, and,
// when a code needs them, details a client can act on. None of them ever
// carries an internal error message, a stack or anything a child wrote. The
// request's id travels in a header beside the body, not inside it.
package apierror

// Response is a structured HTTP error.
type Response struct {
	// Code is a stable identifier a client can branch on.
	Code string `json:"code"`
	// Message is a short sentence describing what happened.
	Message string `json:"message"`
	// Details are machine-readable facts about the error, for the codes that
	// need any. Most do not, and then the field is left out of the body.
	Details map[string]any `json:"details,omitempty"`
}

// The codes the transport itself produces. A code joins this list with the
// first code that answers with it.
const (
	// CodeInternal answers a request the service failed on: a 500.
	CodeInternal = "INTERNAL_ERROR"
	// CodeNotFound answers a path the service does not serve: a 404.
	CodeNotFound = "NOT_FOUND"
	// CodeMethodNotAllowed answers a path asked with a method it does not
	// take: a 405, with the methods it does take in its Allow header.
	CodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
)
