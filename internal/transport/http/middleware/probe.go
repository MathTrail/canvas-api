package middleware

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
// It asks by the route the request is named by in the log, so that the path a
// caller wrote is never read at all, and a line skipped as a probe is one that
// would have named the probe.
func isProbe(route string) bool {
	_, probe := probeRoutes[route]
	return probe
}
