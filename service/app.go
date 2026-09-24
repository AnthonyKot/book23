package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const PublicHost = "api.ledger.example"

type App struct {
	handler     http.Handler
	routes      []Route
	store       *Store
	rateFetcher outboundFetcher
	rateClock   func() time.Time
	rateMode    Mode
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

// NewChapter3App retains Chapters 1 and 2's repairs and isolates the property boundary.
func NewChapter3App(mode Mode) *App {
	return buildAppWithClock(mode, chapter03, time.Now)
}

// NewChapter4App keeps the first three repairs and isolates resource limits.
func NewChapter4App(mode Mode) *App {
	return buildAppWithClock(mode, chapter04, time.Now)
}

// NewChapter5App retains the first four repairs and isolates function access.
func NewChapter5App(mode Mode) *App {
	return buildAppWithClock(mode, chapter05, time.Now)
}

// NewChapter6App retains Chapters 1-5's repairs and isolates refund-flow limits.
func NewChapter6App(mode Mode) *App {
	return buildAppWithClock(mode, chapter06, time.Now)
}

// NewChapter7App retains Chapters 1-6's repairs and isolates outbound fetching.
func NewChapter7App(mode Mode) *App {
	return buildAppWithClock(mode, chapter07, time.Now)
}

// NewChapter8App retains Chapters 1-7's repairs and isolates production settings.
func NewChapter8App(mode Mode) *App {
	return buildAppWithClock(mode, chapter08, time.Now)
}

// NewChapter10App uses Chapters 1-8's repairs and the Chapter 9 fixed gateway policy.
func NewChapter10App(mode Mode) *App {
	return buildAppWithClock(mode, chapter10, time.Now)
}

func (a *App) PollRates(ctx context.Context) error {
	if a.rateClock == nil {
		return fmt.Errorf("rate poller unavailable before Chapter 10")
	}
	if a.rateMode == Vulnerable {
		return a.store.vulnerablePollRates()
	}
	return a.store.fixedPollRates(ctx, a.rateFetcher, a.rateClock())
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
	chapter03
	chapter04
	chapter05
	chapter06
	chapter07
	chapter08
	chapter09
	chapter10
)

type Route struct {
	Method  string
	Pattern string
	Access  Access
}

func buildApp(mode Mode, stage pilotStage) *App {
	return buildAppWithClock(mode, stage, time.Now)
}

func buildAppWithClock(mode Mode, stage pilotStage, clock func() time.Time) *App {
	return buildAppWithNetwork(mode, stage, clock, nil)
}

func buildAppWithNetwork(mode Mode, stage pilotStage, clock func() time.Time, network *chapter07Network) *App {
	return buildAppWithSettings(mode, stage, clock, network, nil)
}

func buildAppWithSettings(mode Mode, stage pilotStage, clock func() time.Time, network *chapter07Network, override *Settings) *App {
	if mode != Vulnerable && mode != Fixed {
		panic("ledger: unknown mode " + mode)
	}
	settings := ProductionSettings()
	if mode == Vulnerable {
		settings = StagingSettings()
	}
	if override != nil {
		settings = *override
	}

	store := seedStore()
	if stage == chapter10 {
		seedChapter10(store, mode, clock())
	}
	mux := http.NewServeMux()
	propertyStage := stage == chapter03 || stage == chapter04 || stage == chapter05 || stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10
	propertyMode := mode
	if stage == chapter04 || stage == chapter05 || stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10 {
		propertyMode = Fixed
	}
	var limits *chapter04Limits
	if stage == chapter04 || stage == chapter05 || stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10 {
		limits = newChapter04Limits(clock)
	}
	limitStage := stage == chapter04 || stage == chapter05 || stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10
	limitMode := mode
	if stage == chapter05 || stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10 {
		limitMode = Fixed
	}
	flowStage := stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10
	flowMode := mode
	if stage == chapter07 || stage == chapter08 || stage == chapter10 {
		flowMode = Fixed
	}
	var fetcher outboundFetcher
	if stage == chapter07 || stage == chapter08 || stage == chapter10 {
		egressMode := mode
		if stage == chapter08 || stage == chapter10 {
			egressMode = Fixed
		}
		fetcher = stage07Fetcher(egressMode, network)
	}
	routes := []Route{{Method: http.MethodGet, Pattern: "/v2/invoices/{id}"}}
	secure := mode == Fixed || stage != chapter01
	registerStageInvoiceRoute(mux, routes[0], store, propertyStage, propertyMode, secure, false)

	listRoute := Route{Method: http.MethodGet, Pattern: "/v2/me/invoices"}
	routes = append(routes, listRoute)
	if propertyStage {
		mux.HandleFunc(listRoute.Method+" "+listRoute.Pattern, chapter03List(store, propertyMode, false))
	} else {
		mux.HandleFunc(listRoute.Method+" "+listRoute.Pattern, listInvoices(store))
	}
	if stage != chapter09 {
		filteredList := Route{Method: http.MethodGet, Pattern: "/v2/invoices"}
		quote := Route{Method: http.MethodPost, Pattern: "/v2/refunds/quote"}
		confirm := Route{Method: http.MethodPost, Pattern: "/v2/refunds/confirm"}
		routes = append(routes, filteredList, quote, confirm)
		if limitStage {
			mux.HandleFunc(filteredList.Method+" "+filteredList.Pattern, chapter04Invoices(store, limitMode, limits))
		} else if propertyStage {
			mux.HandleFunc(filteredList.Method+" "+filteredList.Pattern, chapter03List(store, propertyMode, true))
		} else {
			mux.HandleFunc(filteredList.Method+" "+filteredList.Pattern, listInvoicesWhere(store, secure))
		}
		if stage == chapter10 {
			if mode == Vulnerable {
				mux.HandleFunc(quote.Method+" "+quote.Pattern, vulnerableChapter10Quote(store, clock))
			} else {
				mux.HandleFunc(quote.Method+" "+quote.Pattern, fixedChapter10Quote(store, clock))
			}
		} else if flowStage {
			mux.HandleFunc(quote.Method+" "+quote.Pattern, chapter06Quote(store, clock))
		} else {
			mux.HandleFunc(quote.Method+" "+quote.Pattern, quoteRefund(store))
		}
		confirmMode := mode
		if stage != chapter01 {
			confirmMode = Fixed // later stages retain Chapter 1's refund repair
		}
		if flowStage {
			if flowMode == Vulnerable {
				mux.HandleFunc(confirm.Method+" "+confirm.Pattern, vulnerableChapter06Confirm(store, clock))
			} else {
				mux.HandleFunc(confirm.Method+" "+confirm.Pattern, fixedChapter06Confirm(store, clock))
			}
		} else {
			mux.HandleFunc(confirm.Method+" "+confirm.Pattern, confirmRefund(store, confirmMode))
		}
	}

	serveV1 := stage != chapter09 || mode == Vulnerable
	if serveV1 {
		read := Route{Method: http.MethodGet, Pattern: "/v1/invoices/{id}"}
		pdf := Route{Method: http.MethodGet, Pattern: "/v1/invoices/{id}/pdf"}
		routes = append(routes, read, pdf)
		registerStageInvoiceRoute(mux, read, store, propertyStage, propertyMode, secure, false)
		if stage == chapter07 || stage == chapter08 || stage == chapter10 {
			mux.HandleFunc(pdf.Method+" "+pdf.Pattern, chapter07PDF(store, fetcher))
		} else {
			registerStageInvoiceRoute(mux, pdf, store, propertyStage, propertyMode, secure, true)
		}
		if limitStage {
			lookup := Route{Method: http.MethodGet, Pattern: "/v1/invoices"}
			routes = append(routes, lookup)
			mux.HandleFunc(lookup.Method+" "+lookup.Pattern, chapter04Lookup(store, limitMode, limits, "v1"))
		}
	}

	handler := http.Handler(mux)
	if !propertyStage {
		handler = legacyJSONHandler(handler)
	}
	if stage == chapter02 || propertyStage {
		sessions := newSessionStore(clock)
		login := Route{Method: http.MethodPost, Pattern: "/v2/auth/login"}
		me := Route{Method: http.MethodGet, Pattern: "/v2/me"}
		routes = append(routes, login, me)
		mux.HandleFunc(login.Method+" "+login.Pattern, loginHandler(sessions))
		mux.HandleFunc(me.Method+" "+me.Pattern, identityProbe)
		if propertyStage {
			patchInvoice := Route{Method: http.MethodPatch, Pattern: "/v2/invoices/{id}"}
			patchProfile := Route{Method: http.MethodPatch, Pattern: "/v2/users/me"}
			routes = append(routes, patchInvoice, patchProfile)
			mux.HandleFunc(patchInvoice.Method+" "+patchInvoice.Pattern, chapter03InvoicePatch(store, propertyMode))
			mux.HandleFunc(patchProfile.Method+" "+patchProfile.Pattern, chapter03ProfilePatch(sessions, propertyMode))
		}
		if limitStage {
			requestOTP := Route{Method: http.MethodPost, Pattern: "/v2/auth/otp/request"}
			verifyOTP := Route{Method: http.MethodPost, Pattern: "/v2/auth/otp/verify"}
			routes = append(routes, requestOTP, verifyOTP)
			mux.HandleFunc(requestOTP.Method+" "+requestOTP.Pattern, otpRequestHandler(limitMode, limits.challenges))
			mux.Handle(verifyOTP.Method+" "+verifyOTP.Pattern,
				perIPGuard(limits.otpIP, otpIPBudget, otpVerifyHandler(limitMode, limits.challenges, sessions)))
			handler = chapter04Gateway(limitMode, limits, handler)
		}
		if stage == chapter05 || stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10 {
			accessMode := mode
			if stage == chapter06 || stage == chapter07 || stage == chapter08 || stage == chapter10 {
				accessMode = Fixed
			}
			routes = registerChapter05Routes(mux, routes, store, sessions, accessMode)
			if flowStage {
				routes = registerChapter06Routes(mux, routes, store, flowMode, clock)
			}
			if stage == chapter07 || stage == chapter08 || stage == chapter10 {
				webhookMode := mode
				if stage == chapter08 || stage == chapter10 {
					webhookMode = Fixed
				}
				routes = registerChapter07Routes(mux, routes, webhookMode, fetcher)
			}
			if stage == chapter08 || stage == chapter10 {
				settingsMode := mode
				if stage == chapter10 {
					settingsMode = Fixed
					settings = ProductionSettings()
				}
				routes = registerChapter08Routes(mux, routes, settingsMode, settings)
				if settingsMode == Fixed {
					if err := validateProductionSettings(settings, routes); err != nil {
						panic(err)
					}
				}
			}
			handler = chapter05Authorization(accessMode, mux, routes, store, sessions, handler)
			if stage == chapter08 || stage == chapter10 {
				handler = chapter08ErrorWriter(settings, store, handler)
			}
		}
		identityMode := mode
		if propertyStage {
			identityMode = Fixed
		}
		handler = chapter02Identity(identityMode, sessions, handler)
	}
	if stage == chapter09 {
		handler = chapter09Gateway(mode, handler)
	}
	if stage == chapter10 {
		handler = chapter09Gateway(Fixed, handler)
		previous := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v1/") {
				http.NotFound(w, r)
				return
			}
			previous.ServeHTTP(w, r)
		})
	}
	app := &App{handler: handler, routes: routes, store: store}
	if stage == chapter10 {
		app.rateFetcher, app.rateClock, app.rateMode = fetcher, clock, mode
	}
	return app
}

func registerStageInvoiceRoute(mux *http.ServeMux, route Route, store *Store, propertyStage bool, mode Mode, secure, pdf bool) {
	if propertyStage {
		mux.HandleFunc(route.Method+" "+route.Pattern, chapter03InvoiceHandler(store, mode, pdf))
		return
	}
	registerInvoiceRoute(mux, route, store, secure, pdf)
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
	if _, legacy := w.(legacyResponseWriter); legacy {
		value = legacyJSONValue(value)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "cannot encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append(encoded, '\n'))
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
