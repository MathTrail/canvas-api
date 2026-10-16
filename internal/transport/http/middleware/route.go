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

// knownMethods is the closed set a request's method is reported as. Anything
// else is a caller inventing a verb, and it is reported as one kind.
var knownMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodOptions: {},
	http.MethodConnect: {},
	http.MethodTrace:   {},
}

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

// method is the request's verb when it is one of the known ones.
func method(c *gin.Context) string {
	if _, known := knownMethods[c.Request.Method]; known {
		return c.Request.Method
	}
	return otherLabel
}
