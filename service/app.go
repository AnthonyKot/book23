package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const PublicHost = "api.ledger.example"

type App struct {
	handler http.Handler
	routes  []Route
}

// NewApp constructs the Chapter 1 state. Fixed mode protects /v2 and both
// still-serving /v1 routes; it does not retire them.
func NewApp(mode Mode) *App {
	return buildApp(mode, chapter01)
}

// NewChapter9App constructs the cumulative Chapter 9 state. Its vulnerable
// side begins after the Chapter 1 authorization repair. Its fixed side keeps
// that repair on /v2 while removing /v1 and the staging host.
func NewChapter9App(mode Mode) *App {
	return buildApp(mode, chapter09)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

func (a *App) Routes() []Route {
	return append([]Route(nil), a.routes...)
}

type pilotStage int

const (
	chapter01 pilotStage = iota
	chapter09
)

type Route struct {
	Method  string
	Pattern string
}

func buildApp(mode Mode, stage pilotStage) *App {
	if mode != Vulnerable && mode != Fixed {
		panic("ledger: unknown mode " + mode)
	}

	store := seedStore()
	mux := http.NewServeMux()
	routes := []Route{{Method: http.MethodGet, Pattern: "/v2/invoices/{id}"}}
	secure := mode == Fixed || stage == chapter09
	registerInvoiceRoute(mux, routes[0], store, secure, false)

	listRoute := Route{Method: http.MethodGet, Pattern: "/v2/me/invoices"}
	routes = append(routes, listRoute)
	mux.HandleFunc(listRoute.Method+" "+listRoute.Pattern, listInvoices(store))

	serveV1 := stage == chapter01 || mode == Vulnerable
	if serveV1 {
		read := Route{Method: http.MethodGet, Pattern: "/v1/invoices/{id}"}
		pdf := Route{Method: http.MethodGet, Pattern: "/v1/invoices/{id}/pdf"}
		routes = append(routes, read, pdf)
		registerInvoiceRoute(mux, read, store, secure, false)
		registerInvoiceRoute(mux, pdf, store, secure, true)
	}

	handler := http.Handler(mux)
	if stage == chapter09 {
		handler = chapter09Gateway(mode, mux)
	}
	return &App{handler: handler, routes: routes}
}

func registerInvoiceRoute(mux *http.ServeMux, route Route, store *Store, secure, pdf bool) {
	var handler http.HandlerFunc
	if secure {
		handler = fixedInvoiceHandler(store, pdf)
	} else {
		handler = vulnerableInvoiceHandler(store, pdf)
	}
	mux.HandleFunc(route.Method+" "+route.Pattern, handler)
}

// excerpt: ch01-vulnerable-invoice-handler
func vulnerableInvoiceHandler(store *Store, pdf bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentUser(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.Invoice(id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		writeInvoice(w, invoice, pdf)
	}
}

// end excerpt

// excerpt: ch01-fixed-invoice-handler
func fixedInvoiceHandler(store *Store, pdf bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.LoadInvoiceFor(user, id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		writeInvoice(w, invoice, pdf)
	}
}

// end excerpt

// excerpt: ch01-safe-list-decoy
func listInvoices(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, store.InvoicesFor(user))
	}
}

// end excerpt

func currentUser(r *http.Request) (User, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return User{}, false
	}
	user, ok := usersByToken[strings.TrimPrefix(header, prefix)]
	return user, ok
}

func writeInvoice(w http.ResponseWriter, invoice Invoice, pdf bool) {
	if pdf {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Ledger PDF invoice=%d tenant=%s amount=%d\n", invoice.ID, invoice.Tenant, invoice.Amount)
		return
	}
	writeJSON(w, invoice)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// excerpt: ch09-gateway-host-filter
func chapter09Gateway(mode Mode, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == PublicHost {
			next.ServeHTTP(w, r)
			return
		}
		if mode == Vulnerable && host == StagingHost && strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// end excerpt
