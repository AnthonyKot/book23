package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const PublicHost = "api.ledger.example"

type App struct {
	handler http.Handler
	routes  []Route
	store   *Store
}

// NewApp constructs the Chapter 1 state. Fixed mode protects /v2 and both
// still-serving /v1 routes; it does not retire them.
func NewApp(mode Mode) *App {
	return buildApp(mode, chapter01)
}

// NewChapter2App keeps Chapter 1's loader repair in both modes. Its modes
// differ only in how a credential becomes the caller's identity.
func NewChapter2App(mode Mode) *App {
	return newChapter2AppAt(mode, time.Now)
}

// NewChapter9App is the Chapter 9 pilot. It includes Chapter 1's loader repair,
// but not yet the intervening Chapters 2-8. Its fixed side removes /v1 and the
// staging host; batch 2 integrates the intervening repairs after Chapter 8.
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
	chapter02
	chapter09
)

type Route struct {
	Method  string
	Pattern string
}

func buildApp(mode Mode, stage pilotStage) *App {
	return buildAppWithClock(mode, stage, time.Now)
}

func buildAppWithClock(mode Mode, stage pilotStage, clock func() time.Time) *App {
	if mode != Vulnerable && mode != Fixed {
		panic("ledger: unknown mode " + mode)
	}

	store := seedStore()
	mux := http.NewServeMux()
	routes := []Route{{Method: http.MethodGet, Pattern: "/v2/invoices/{id}"}}
	secure := mode == Fixed || stage != chapter01
	registerInvoiceRoute(mux, routes[0], store, secure, false)

	listRoute := Route{Method: http.MethodGet, Pattern: "/v2/me/invoices"}
	routes = append(routes, listRoute)
	mux.HandleFunc(listRoute.Method+" "+listRoute.Pattern, listInvoices(store))
	if stage != chapter09 {
		filteredList := Route{Method: http.MethodGet, Pattern: "/v2/invoices"}
		quote := Route{Method: http.MethodPost, Pattern: "/v2/refunds/quote"}
		confirm := Route{Method: http.MethodPost, Pattern: "/v2/refunds/confirm"}
		routes = append(routes, filteredList, quote, confirm)
		mux.HandleFunc(filteredList.Method+" "+filteredList.Pattern, listInvoicesWhere(store, secure))
		mux.HandleFunc(quote.Method+" "+quote.Pattern, quoteRefund(store))
		confirmMode := mode
		if stage != chapter01 {
			confirmMode = Fixed // later stages retain Chapter 1's refund repair
		}
		mux.HandleFunc(confirm.Method+" "+confirm.Pattern, confirmRefund(store, confirmMode))
	}

	serveV1 := stage != chapter09 || mode == Vulnerable
	if serveV1 {
		read := Route{Method: http.MethodGet, Pattern: "/v1/invoices/{id}"}
		pdf := Route{Method: http.MethodGet, Pattern: "/v1/invoices/{id}/pdf"}
		routes = append(routes, read, pdf)
		registerInvoiceRoute(mux, read, store, secure, false)
		registerInvoiceRoute(mux, pdf, store, secure, true)
	}

	handler := http.Handler(mux)
	if stage == chapter02 {
		sessions := newSessionStore(clock)
		login := Route{Method: http.MethodPost, Pattern: "/v2/auth/login"}
		me := Route{Method: http.MethodGet, Pattern: "/v2/me"}
		routes = append(routes, login, me)
		mux.HandleFunc(login.Method+" "+login.Pattern, loginHandler(sessions))
		mux.HandleFunc(me.Method+" "+me.Pattern, identityProbe)
		handler = chapter02Identity(mode, sessions, mux)
	}
	if stage == chapter09 {
		handler = chapter09Gateway(mode, mux)
	}
	return &App{handler: handler, routes: routes, store: store}
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
	if principal, ok := r.Context().Value(principalContextKey{}).(requestPrincipal); ok {
		return principal.user, principal.authenticated
	}
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
