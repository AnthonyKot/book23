package ledger

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type chapter11Caller struct{ name, token, tenant, role string }

var chapter11Callers = []chapter11Caller{
	{name: "anonymous"},
	{name: "Alice", token: "alice-token", tenant: "Cedar", role: "user"},
	{name: "Ben", token: "ben-token", tenant: "Birch", role: "user"},
	{name: "Dana", token: "dana-token", tenant: "Cedar", role: "tenant-admin"},
}

type chapter11Case struct {
	path, body, targetTenant string
	allowedStatus            int
	exemption                string
}

func chapter11App() *App {
	rt := &chapter07RoundTrip{reply: func(*http.Request) (int, string, string) { return 200, "", "ok" }}
	answers := map[string][]netip.Addr{"hooks.cedar.example": {netip.MustParseAddr("8.8.8.8")}}
	queries := []string{}
	fixed := &egress{resolve: chapter07Resolver(answers, &queries),
		dial: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("canned transport should not dial")
		},
		transport: rt}
	app := buildAppWithNetwork(Fixed, chapter10, func() time.Time { return chapter06TestTime },
		&chapter07Network{vulnerable: defaultFetch{&http.Client{Transport: rt}}, fixed: fixed})
	inv, _ := app.store.Invoice(104)
	app.store.newQuote(usersByToken["alice-token"], inv, 40000) // q-771 for the approval/confirm cells
	return app
}

// excerpt: ch11-every-route-every-caller
func TestChapter11EveryRouteEveryCaller(t *testing.T) {
	routes := NewChapter10App(Fixed).Routes()
	if len(routes) != 20 {
		t.Fatalf("route inventory changed: %d", len(routes))
	}
	for _, route := range routes {
		for _, cell := range chapter11ConcreteRequests(t, route) {
			for _, caller := range chapter11Callers {
				t.Run(route.Method+" "+cell.path+"/"+caller.name, func(t *testing.T) {
					app := chapter11App()
					path := cell.path
					if route.Pattern == "/v2/invoices" && caller.tenant != "" {
						path += "?tenant=" + caller.tenant
					}
					got := chapter04Request(t, app, route.Method, path, caller.token, "192.0.2.44", cell.body)
					chapter11AssertTenant(t, caller, got)
					chapter11AssertNoInternalFields(t, got)
					chapter11AssertOpaqueInvoice404(t, route, got)
					chapter11AssertDeclaredStatus(t, route, cell, caller, got)
				})
			}
		}
	}
}

// end excerpt

func chapter11ConcreteRequests(t *testing.T, route Route) []chapter11Case {
	t.Helper()
	key := route.Method + " " + route.Pattern
	invoicePair := func(body string, allowed int) []chapter11Case {
		return []chapter11Case{
			{path: strings.Replace(route.Pattern, "{id}", "104", 1), body: body, targetTenant: "Cedar", allowedStatus: allowed},
			{path: strings.Replace(route.Pattern, "{id}", "205", 1), body: body, targetTenant: "Birch", allowedStatus: allowed},
		}
	}
	switch key {
	case "GET /v2/invoices/{id}":
		return invoicePair("", 200)
	case "PATCH /v2/invoices/{id}":
		return invoicePair(`{"reference":"PO-9"}`, 200)
	case "DELETE /v2/invoices/{id}":
		return invoicePair("", 204)
	case "POST /v2/admin/invoices/{id}/void":
		return invoicePair("", 204)
	case "POST /v2/invoices/{id}/remind":
		return invoicePair("", 200)
	case "PATCH /v2/admin/users/{id}":
		return []chapter11Case{{path: "/v2/admin/users/alice", body: `{"role":"user"}`, targetTenant: "Cedar", allowedStatus: 200},
			{path: "/v2/admin/users/ben", body: `{"role":"user"}`, targetTenant: "Birch", allowedStatus: 200}}
	case "POST /v2/refunds/{quote}/approve":
		return []chapter11Case{{path: "/v2/refunds/q-771/approve", targetTenant: "Cedar", allowedStatus: 400, exemption: "open quote is not parked for approval"}}
	}
	// These are the only minimal-body exceptions: an OTP cannot be verified before
	// a challenge exists, and an open refund quote cannot be approved.
	requests := map[string]chapter11Case{
		"GET /v2/me/invoices":       {path: "/v2/me/invoices", allowedStatus: 200},
		"GET /v2/invoices":          {path: "/v2/invoices", allowedStatus: 200},
		"POST /v2/refunds/quote":    {path: "/v2/refunds/quote", body: `{"invoice_id":104,"amount":40000}`, targetTenant: "Cedar", allowedStatus: 200},
		"POST /v2/refunds/confirm":  {path: "/v2/refunds/confirm", body: `{"quote_id":"q-771"}`, targetTenant: "Cedar", allowedStatus: 200},
		"POST /v2/auth/login":       {path: "/v2/auth/login", body: `{"user":"alice","password":"fixture-alice"}`, allowedStatus: 200},
		"GET /v2/me":                {path: "/v2/me", allowedStatus: 200},
		"PATCH /v2/users/me":        {path: "/v2/users/me", body: `{"display_name":"A"}`, allowedStatus: 200},
		"POST /v2/auth/otp/request": {path: "/v2/auth/otp/request", body: `{"account_id":"Ben"}`, allowedStatus: 200},
		"POST /v2/auth/otp/verify":  {path: "/v2/auth/otp/verify", body: `{"account_id":"Ben","challenge_id":"otp-missing","code":"000000"}`, allowedStatus: 400, exemption: "no active OTP challenge"},
		"GET /v2/admin/users":       {path: "/v2/admin/users", allowedStatus: 200},
		"POST /v2/webhooks/test":    {path: "/v2/webhooks/test", body: `{"url":"https://hooks.cedar.example/ledger"}`, allowedStatus: 200},
		"GET /v2/health":            {path: "/v2/health", allowedStatus: 200},
		"GET /v2/admin/health":      {path: "/v2/admin/health", allowedStatus: 200},
	}
	cell, ok := requests[key]
	if !ok {
		t.Fatalf("route has no concrete request: %s", key)
	}
	return []chapter11Case{cell}
}

func chapter11AssertTenant(t *testing.T, caller chapter11Caller, got response) {
	t.Helper()
	var forbidden []string
	switch caller.tenant {
	case "Cedar":
		forbidden = []string{"Birch", "Ben", "205", "LGR-K8", "birch-billing@example.test"}
	case "Birch":
		forbidden = []string{"Cedar", "Alice", "Dana", "104", "LGR-A7", "LGR-E4", "cedar-billing@example.test"}
	default:
		forbidden = []string{"Cedar", "Birch", "Alice", "Ben", "Dana", "LGR-A7", "LGR-K8", "LGR-E4"}
	}
	for _, value := range forbidden {
		if strings.Contains(got.body, value) {
			t.Fatalf("%s response exposes %q: %+v", caller.name, value, got)
		}
	}
}

func chapter11AssertNoInternalFields(t *testing.T, got response) {
	t.Helper()
	body := strings.ToLower(got.body)
	for _, field := range []string{"collections_note", "collectionsnote", "margin"} {
		if strings.Contains(body, field) {
			t.Fatalf("internal field %q in response: %+v", field, got)
		}
	}
}

func chapter11AssertOpaqueInvoice404(t *testing.T, route Route, got response) {
	t.Helper()
	if route.Method == http.MethodGet && route.Pattern == "/v2/invoices/{id}" && got.status == 404 && got.body != "{\"error\":\"not found\"}\n" {
		t.Fatalf("invoice denial body = %+v", got)
	}
}

func chapter11AssertDeclaredStatus(t *testing.T, route Route, cell chapter11Case, caller chapter11Caller, got response) {
	t.Helper()
	want := cell.allowedStatus
	if caller.tenant == "" && route.Access != Public {
		want = 401
	} else if cell.targetTenant != "" && caller.tenant != "" && caller.tenant != cell.targetTenant {
		want = 404 // scoped object, quote, or admin user precedes role
	} else if route.Access == TenantAdmin && caller.role != "tenant-admin" {
		want = 403
	}
	if got.status != want {
		t.Fatalf("%s %s: status=%d want=%d body=%q", caller.name, route.Method, got.status, want, got.body)
	}
	if got.status == 400 && cell.exemption == "" {
		t.Fatalf("unexpected 400 without named exemption: %s %s", route.Method, cell.path)
	}
	if route.Access == Public && caller.tenant == "" && (got.status == 401 || got.status == 403) {
		t.Fatalf("public route refused anonymous: %+v", got)
	}
}

func TestChapter11ExerciseLayers(t *testing.T) {
	app := chapter11App()
	if got := chapter05Request(t, app, http.MethodGet, "/v2/health", "", ""); got.status != 200 || got.body != "{\"ok\":true}\n" {
		t.Fatalf("A health: %+v", got)
	}
	if got := chapter05Request(t, app, http.MethodGet, "/v2/admin/health", "", ""); got.status != 401 {
		t.Fatalf("B anonymous admin: %+v", got)
	}
	if got := chapter05Request(t, app, http.MethodGet, "/v2/admin/health", "dana-token", ""); got.status != 200 || got.body != "{\"version\":\"fixture\"}\n" {
		t.Fatalf("B Dana: %+v", got)
	}
	if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/205", "alice-token", ""); got.status != 404 || got.body != "{\"error\":\"not found\"}\n" {
		t.Fatalf("C scope: %+v", got)
	}
	if got := chapter05Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", ""); got.status != 403 || !hasInvoice(app.store, 104) {
		t.Fatalf("D role: %+v", got)
	}
	if got := chapter05Request(t, app, http.MethodDelete, "/v2/invoices/205", "dana-token", ""); got.status != 404 || !hasInvoice(app.store, 205) {
		t.Fatalf("E scope: %+v", got)
	}
	for i := 0; i < 30; i++ {
		got := chapter04Request(t, app, http.MethodGet, "/v2/invoices?email=x@cedar.example", "alice-token", "192.0.2.44", "")
		if got.status != 200 {
			t.Fatalf("F lookup %d: %+v", i, got)
		}
	}
	for _, ip := range []string{"192.0.2.44", "192.0.2.99"} {
		got := chapter04Request(t, app, http.MethodGet, "/v2/invoices?email=x@cedar.example", "alice-token", ip, "")
		if got.status != 429 || !strings.Contains(got.body, "gateway lookup limit") {
			t.Fatalf("F 31st from %s: %+v", ip, got)
		}
	}
	staging := httptest.NewRequest(http.MethodGet, "http://"+StagingHost+"/v2/invoices/104", nil)
	staging.Host = StagingHost
	staging.Header.Set("Authorization", "Bearer garbage-token")
	reply := httptest.NewRecorder()
	app.ServeHTTP(reply, staging)
	if reply.Code != 404 {
		t.Fatalf("G staging host: %d %q", reply.Code, reply.Body.String())
	}
	staging.Header.Set("Authorization", "Bearer alice-token")
	validReply := httptest.NewRecorder()
	app.ServeHTTP(validReply, staging)
	if validReply.Code != reply.Code || validReply.Body.String() != reply.Body.String() {
		t.Fatalf("G staging host leaked identity: valid=%d %q invalid=%d %q", validReply.Code, validReply.Body.String(), reply.Code, reply.Body.String())
	}
	if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/104", "alice-token", ""); got.status != 200 {
		t.Fatalf("G public host: %+v", got)
	}
	if got := chapter05Request(t, app, http.MethodPost, "/v2/webhooks/test", "alice-token", `{"url":"https://hooks.cedar.example/ledger"}`); got.status != 200 {
		t.Fatalf("H public egress baseline: %+v", got)
	}
	// H's link-local answer is tested with a separate resolver so no dial is possible.
	unsafe := chapter11UnsafeEgressApp(t)
	if got := chapter05Request(t, unsafe, http.MethodPost, "/v2/webhooks/test", "alice-token", `{"url":"https://hooks.cedar.example/ledger"}`); got.status != 400 {
		t.Fatalf("H link-local: %+v", got)
	}
}

func chapter11UnsafeEgressApp(t *testing.T) *App {
	t.Helper()
	queries := []string{}
	rt := &chapter07RoundTrip{reply: func(*http.Request) (int, string, string) {
		t.Fatal("H unexpectedly reached transport")
		return 0, "", ""
	}}
	fixed := &egress{resolve: chapter07Resolver(map[string][]netip.Addr{"hooks.cedar.example": {netip.MustParseAddr("169.254.169.254")}}, &queries),
		dial: func(context.Context, string, string) (net.Conn, error) {
			t.Fatal("H unexpectedly dialed")
			return nil, errors.New("unexpected dial")
		}, transport: rt}
	return buildAppWithNetwork(Fixed, chapter10, func() time.Time { return chapter06TestTime },
		&chapter07Network{vulnerable: defaultFetch{&http.Client{Transport: rt}}, fixed: fixed})
}
