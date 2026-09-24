package ledger

import (
	"net/http"
	"strings"
)

// The old staging rule missed the gateway copy of Chapter 4's lookup budget.
// The shared lookup handler still applies its tenant-and-route budget.
func chapter09LookupGateway(mode Mode, limits *chapter04Limits, next http.Handler) http.Handler {
	metered := chapter04Gateway(Fixed, limits, next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode == Vulnerable && r.Host == StagingHost || mode == Fixed && strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		metered.ServeHTTP(w, r)
	})
}
