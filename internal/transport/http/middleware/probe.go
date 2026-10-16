package middleware

import "github.com/gin-gonic/gin"

// probeRoutes answer a machine rather than a person. The platform asks for them
// constantly and the answer is the same every time.
var probeRoutes = map[string]struct{}{
	"/health": {},
}

// isProbe reports whether a request matched one of those.
//
// The request log asks, and leaves a probe out: there are far more of them than
// there are checks, so a probe in the log buries the lines that mean
// something.
//
// It asks by the route, as everything else here names a request, so that the
// path a caller wrote is never read at all.
func isProbe(c *gin.Context) bool {
	_, probe := probeRoutes[route(c)]
	return probe
}
