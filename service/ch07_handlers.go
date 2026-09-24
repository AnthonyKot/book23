package ledger

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

func chapter07WebhookInput(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input struct {
		URL string `json:"url"`
	}
	if decodeOneJSON(r, &input) != nil {
		http.Error(w, "invalid webhook test", http.StatusBadRequest)
		return "", false
	}
	u, err := url.Parse(input.URL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || input.URL == "http://169.254.169.254/" {
		http.Error(w, "invalid webhook URL", http.StatusBadRequest)
		return "", false
	}
	return input.URL, true
}

func writeWebhookResult(w http.ResponseWriter, reply *http.Response, err error) {
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errEgressDenied) {
			status = http.StatusBadRequest
		}
		http.Error(w, "outbound fetch refused", status)
		return
	}
	defer reply.Body.Close()
	writeJSON(w, struct {
		Status int `json:"status"`
	}{reply.StatusCode})
}

// excerpt: ch07-webhook-vulnerable
func vulnerableChapter07Webhook(fetcher outboundFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawURL, ok := chapter07WebhookInput(w, r) // checks only the submitted URL
		if !ok {
			return
		}
		reply, err := fetcher.Fetch(r.Context(), rawURL) // default client resolves and redirects
		writeWebhookResult(w, reply, err)
	}
}

// end excerpt

// excerpt: ch07-webhook-fixed
func fixedChapter07Webhook(fetcher outboundFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawURL, ok := chapter07WebhookInput(w, r)
		if !ok {
			return
		}
		reply, err := fetcher.Fetch(r.Context(), rawURL) // shared egress checks the socket target
		writeWebhookResult(w, reply, err)
	}
}

// end excerpt

// excerpt: ch07-pdf-logo-fetch
func chapter07PDF(store *Store, fetcher outboundFetcher) http.HandlerFunc {
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
		logo := store.branding[invoice.Tenant].LogoURL
		if logo != "" {
			reply, err := fetcher.Fetch(r.Context(), logo)
			if err == nil {
				reply.Body.Close()
			}
		}
		renderViewPDF(w, publicViewFor(invoice)) // unavailable logo never exposes a row
	}
}

// end excerpt

func registerChapter07Routes(mux *http.ServeMux, routes []Route, mode Mode, fetcher outboundFetcher) []Route {
	route := Route{Method: http.MethodPost, Pattern: "/v2/webhooks/test"}
	if mode == Vulnerable {
		mux.HandleFunc(route.Method+" "+route.Pattern, vulnerableChapter07Webhook(fetcher))
	} else {
		mux.HandleFunc(route.Method+" "+route.Pattern, fixedChapter07Webhook(fetcher))
	}
	policy := append(append([]Route{}, chapter05Declarations...), chapter06Declarations...)
	policy = append(policy, chapter07Declarations...)
	return declareRoutes(append(routes, route), policy)
}
