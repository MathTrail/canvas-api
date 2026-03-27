// Package apierror defines shared HTTP error response types.
package apierror

// Response represents a structured HTTP error response.
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
