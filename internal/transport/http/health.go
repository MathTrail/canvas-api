package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MathTrail/canvas-api/internal/version"
)

// healthResponse is the body of the probe. It is a type rather than a map so
// that the shape of the answer is stated once, in one place, and a mistyped
// key is a compile error instead of a field nobody notices is missing.
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// health says the process is up and serving, and which build it is: the
// delivery reads the commit here to know that the revision it rolled out is
// the one that answers, so no cache on the way may answer for it. It checks
// nothing else on purpose: a probe that reached for a dependency would turn
// that dependency's outage into restarts of this service.
func health(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, healthResponse{
		Status:  "ok",
		Version: version.Version,
		Commit:  version.Commit,
	})
}
