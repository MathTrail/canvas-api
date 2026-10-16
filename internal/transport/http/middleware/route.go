package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// otherLabel stands for anything outside the closed sets this service wrote
// down, the ones below, which a request is called by in a line of the log. It
// says that something was there, and nothing of what it was: a log line that
// carries what a caller wrote carries whatever they chose to write there.
const otherLabel = "other"

// route is the pattern the request matched, never the path it asked for. A
// path carries identifiers, one per sheet or task a log line would then
// spread; and a path nothing matched is text a caller wrote, as able to carry
// a name as anything else they write.
//
// The framework knows the pattern before the first handler runs, so every
// middleware can ask, a panicking one included.
func route(c *gin.Context) string {
	if matched := c.FullPath(); matched != "" {
		return matched
	}
	return otherLabel
}

// method is the request's verb when it is one of the known ones, and other
// otherwise: anything else is a caller inventing a verb, and it is reported as
// one kind. The set is closed, so the word written is always one of the
// service's own, spelled here.
func method(c *gin.Context) string {
	switch c.Request.Method {
	case http.MethodGet:
		return http.MethodGet
	case http.MethodHead:
		return http.MethodHead
	case http.MethodPost:
		return http.MethodPost
	case http.MethodPut:
		return http.MethodPut
	case http.MethodPatch:
		return http.MethodPatch
	case http.MethodDelete:
		return http.MethodDelete
	case http.MethodOptions:
		return http.MethodOptions
	case http.MethodConnect:
		return http.MethodConnect
	case http.MethodTrace:
		return http.MethodTrace
	default:
		return otherLabel
	}
}
