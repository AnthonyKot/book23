package ledger

import "net/http"

// excerpt: ch11-request-path
func chapter11RequestPath(mux *http.ServeMux, routes []Route, store *Store,
	sessions *sessionStore, limits *chapter04Limits, settings Settings) http.Handler {
	var handler http.Handler = mux
	handler = chapter09LookupGateway(Fixed, limits, handler)
	handler = chapter05Authorization(Fixed, mux, routes, store, sessions, handler)
	handler = chapter08ErrorWriter(settings, store, handler)
	handler = chapter02Identity(Fixed, sessions, handler)
	handler = chapter09Gateway(Fixed, handler)
	return handler
}

// end excerpt
