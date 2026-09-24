package ledger

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type response struct {
	status int
	body   string
}

func request(t *testing.T, handler http.Handler, host, token, path string) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
	req.Host = host
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	result := recorder.Result()
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response{status: result.StatusCode, body: string(body)}
}

type chapter01Expectation struct {
	status  int
	include string
	exclude string
}

type chapter01ReadCase struct {
	name, token, path string
	route             Route
	vulnerable, fixed chapter01Expectation
}

var chapter01ReadCases = []chapter01ReadCase{
	{"v2 Alice own invoice", "alice-token", "/v2/invoices/104", Route{Method: "GET", Pattern: "/v2/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}},
	{"v2 Alice Birch invoice", "alice-token", "/v2/invoices/205", Route{Method: "GET", Pattern: "/v2/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Birch"`, ""}, chapter01Expectation{404, "", "Birch"}},
	{"v2 Ben Cedar invoice", "ben-token", "/v2/invoices/104", Route{Method: "GET", Pattern: "/v2/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{404, "", "Cedar"}},
	{"v1 Alice own invoice", "alice-token", "/v1/invoices/104", Route{Method: "GET", Pattern: "/v1/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}},
	{"v1 Alice Birch invoice", "alice-token", "/v1/invoices/205", Route{Method: "GET", Pattern: "/v1/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Birch"`, ""}, chapter01Expectation{404, "", "Birch"}},
	{"v1 Ben Cedar invoice", "ben-token", "/v1/invoices/104", Route{Method: "GET", Pattern: "/v1/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{404, "", "Cedar"}},
	{"PDF Alice own invoice", "alice-token", "/v1/invoices/104/pdf", Route{Method: "GET", Pattern: "/v1/invoices/{id}/pdf"}, chapter01Expectation{200, "tenant=Cedar", ""}, chapter01Expectation{200, "tenant=Cedar", ""}},
	{"PDF Alice Birch invoice", "alice-token", "/v1/invoices/205/pdf", Route{Method: "GET", Pattern: "/v1/invoices/{id}/pdf"}, chapter01Expectation{200, "tenant=Birch", ""}, chapter01Expectation{404, "", "Birch"}},
	{"PDF Ben Cedar invoice", "ben-token", "/v1/invoices/104/pdf", Route{Method: "GET", Pattern: "/v1/invoices/{id}/pdf"}, chapter01Expectation{200, "tenant=Cedar", ""}, chapter01Expectation{404, "", "Cedar"}},
	{"safe list decoy Alice", "alice-token", "/v2/me/invoices", Route{Method: "GET", Pattern: "/v2/me/invoices"}, chapter01Expectation{200, `"id":104`, `"id":205`}, chapter01Expectation{200, `"id":104`, `"id":205`}},
	{"safe list decoy Ben", "ben-token", "/v2/me/invoices", Route{Method: "GET", Pattern: "/v2/me/invoices"}, chapter01Expectation{200, `"id":205`, `"id":104`}, chapter01Expectation{200, `"id":205`, `"id":104`}},
	{"filtered list Alice asks Birch", "alice-token", "/v2/invoices?tenant=birch", Route{Method: "GET", Pattern: "/v2/invoices"}, chapter01Expectation{200, `"id":205`, `"id":104`}, chapter01Expectation{200, `[]`, `"id":205`}},
	{"filtered list Alice asks Cedar", "alice-token", "/v2/invoices?tenant=cedar", Route{Method: "GET", Pattern: "/v2/invoices"}, chapter01Expectation{200, `"id":104`, `"id":205`}, chapter01Expectation{200, `"id":104`, `"id":205`}},
	{"filtered list Ben asks Cedar", "ben-token", "/v2/invoices?tenant=cedar", Route{Method: "GET", Pattern: "/v2/invoices"}, chapter01Expectation{200, `"id":104`, `"id":205`}, chapter01Expectation{200, `[]`, `"id":104`}},
	{"filtered list Ben asks Birch", "ben-token", "/v2/invoices?tenant=birch", Route{Method: "GET", Pattern: "/v2/invoices"}, chapter01Expectation{200, `"id":205`, `"id":104`}, chapter01Expectation{200, `"id":205`, `"id":104`}},
	{"Dana Cedar admin", "dana-token", "/v2/invoices/104", Route{Method: "GET", Pattern: "/v2/invoices/{id}"}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{200, `"tenant":"Cedar"`, ""}},
}

func TestChapter01ExerciseCases(t *testing.T) {

	apps := map[Mode]*App{Vulnerable: NewApp(Vulnerable), Fixed: NewApp(Fixed)}
	for _, test := range chapter01ReadCases {
		for _, mode := range []Mode{Vulnerable, Fixed} {
			t.Run(string(mode)+"/"+test.name, func(t *testing.T) {
				want := test.vulnerable
				if mode == Fixed {
					want = test.fixed
				}
				got := request(t, apps[mode], PublicHost, test.token, test.path)
				if got.status != want.status {
					t.Fatalf("status = %d, want %d; body=%q", got.status, want.status, got.body)
				}
				if want.include != "" && !strings.Contains(got.body, want.include) {
					t.Errorf("body %q does not include %q", got.body, want.include)
				}
				if want.exclude != "" && strings.Contains(got.body, want.exclude) {
					t.Errorf("body %q unexpectedly includes %q", got.body, want.exclude)
				}
			})
		}
	}
}

func TestChapter01RegisteredInvoiceRoutesCovered(t *testing.T) {
	covered := make(map[Route]map[string]bool)
	for _, test := range chapter01ReadCases {
		if covered[test.route] == nil {
			covered[test.route] = make(map[string]bool)
		}
		covered[test.route][test.token] = true
	}
	for _, mode := range []Mode{Vulnerable, Fixed} {
		registered := make(map[Route]bool)
		for _, route := range NewApp(mode).Routes() {
			if route.Method == http.MethodGet {
				registered[route] = true
			}
		}
		for route := range registered {
			if !covered[route]["alice-token"] || !covered[route]["ben-token"] {
				t.Errorf("%s registered invoice route lacks Alice/Ben read cases: %+v", mode, route)
			}
		}
		for route := range covered {
			if !registered[route] {
				t.Errorf("%s tested invoice route is not registered: %+v", mode, route)
			}
		}
	}
}

func postJSON(t *testing.T, handler http.Handler, token, path, body string) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://"+PublicHost+path, strings.NewReader(body))
	req.Host = PublicHost
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	result := recorder.Result()
	defer result.Body.Close()
	content, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response{status: result.StatusCode, body: string(content)}
}

func firstQuote(t *testing.T, app *App) Quote {
	t.Helper()
	got := postJSON(t, app, "alice-token", "/v2/refunds/quote", `{"invoice_id":104,"amount":40000}`)
	if got.status != 200 {
		t.Fatalf("quote status = %d; body=%q", got.status, got.body)
	}
	var quote Quote
	if err := json.Unmarshal([]byte(got.body), &quote); err != nil {
		t.Fatal(err)
	}
	if quote.ID != "q-771" || quote.InvoiceID != 104 || quote.Amount != 40000 {
		t.Fatalf("first quote = %+v", quote)
	}
	return quote
}

func TestChapter01RefundSequence(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode)+"/injected confirm", func(t *testing.T) {
			app := NewApp(mode)
			if got := postJSON(t, app, "ben-token", "/v2/refunds/quote", `{"invoice_id":104,"amount":40000}`); got.status != 404 || len(app.store.quotes) != 0 {
				t.Fatalf("Ben quoting Cedar invoice = %+v; quotes=%+v", got, app.store.quotes)
			}
			firstQuote(t, app)
			if got := postJSON(t, app, "ben-token", "/v2/refunds/confirm", `{"quote_id":"q-771"}`); got.status != 404 {
				t.Fatalf("Ben confirming Alice's quote = %d; body=%q", got.status, got.body)
			}
			got := postJSON(t, app, "alice-token", "/v2/refunds/confirm", `{"quote_id":"q-771","invoice_id":205}`)
			cedar, _ := app.store.Invoice(104)
			birch, _ := app.store.Invoice(205)
			if mode == Vulnerable {
				if got.status != 200 || !strings.Contains(got.body, `"invoice_id":205`) ||
					birch.Refunded != 40000 || cedar.Refunded != 0 || len(app.store.refunds) != 1 {
					t.Fatalf("vulnerable confirm = %+v; Cedar=%+v Birch=%+v refunds=%+v", got, cedar, birch, app.store.refunds)
				}
			} else {
				if got.status != 400 || len(app.store.refunds) != 0 || cedar.Refunded != 0 || birch.Refunded != 0 {
					t.Fatalf("fixed confirm = %+v; Cedar=%+v Birch=%+v refunds=%+v", got, cedar, birch, app.store.refunds)
				}
				withNull := postJSON(t, app, "alice-token", "/v2/refunds/confirm", `{"quote_id":"q-771","invoice_id":null}`)
				if withNull.status != 400 || len(app.store.refunds) != 0 {
					t.Fatalf("fixed confirm accepted extra null invoice_id: %+v", withNull)
				}
			}
		})

		t.Run(string(mode)+"/valid stored target", func(t *testing.T) {
			app := NewApp(mode)
			firstQuote(t, app)
			body := `{"quote_id":"q-771"}`
			first := postJSON(t, app, "alice-token", "/v2/refunds/confirm", body)
			second := postJSON(t, app, "alice-token", "/v2/refunds/confirm", body)
			if first.status != 200 || second.status != 200 || first.body != second.body {
				t.Fatalf("confirm/replay = %+v / %+v", first, second)
			}
			want := Refund{QuoteID: "q-771", InvoiceID: 104, Tenant: "Cedar", Amount: 40000}
			if !reflect.DeepEqual(app.store.refunds, []Refund{want}) {
				t.Fatalf("stored refunds = %+v, want %+v", app.store.refunds, want)
			}
			cedar, _ := app.store.Invoice(104)
			birch, _ := app.store.Invoice(205)
			if cedar.Refunded != 40000 || birch.Refunded != 0 {
				t.Fatalf("refund state: Cedar=%+v Birch=%+v", cedar, birch)
			}
		})

		t.Run(string(mode)+"/first 104 quote keeps fixed ID", func(t *testing.T) {
			app := NewApp(mode)
			other := postJSON(t, app, "ben-token", "/v2/refunds/quote", `{"invoice_id":205,"amount":40000}`)
			if other.status != 200 || strings.Contains(other.body, `"quote_id":"q-771"`) {
				t.Fatalf("Ben's quote consumed Alice's fixed ID: %+v", other)
			}
			firstQuote(t, app)
		})
	}
}
