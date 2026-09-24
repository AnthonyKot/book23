package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestChapter09ExerciseCases(t *testing.T) {
	tests := []struct {
		name, host, path string
		vulnerableStatus int
		fixedStatus      int
	}{
		{"current public route decoy", PublicHost, "/v2/invoices/104", 200, 200},
		{"current route keeps Chapter 1 loader", PublicHost, "/v2/invoices/205", 404, 404},
		{"deprecated public route", PublicHost, "/v1/invoices/104", 200, 404},
		{"unlisted staging v1 route", StagingHost, "/v1/invoices/104", 200, 404},
		{"near-identical staging v2 route", StagingHost, "/v2/invoices/104", 404, 404},
		{"unknown preview host decoy", "ledger-preview.internal", "/v1/invoices/104", 404, 404},
	}

	apps := map[Mode]*App{Vulnerable: NewChapter9App(Vulnerable), Fixed: NewChapter9App(Fixed)}
	for _, test := range tests {
		for _, mode := range []Mode{Vulnerable, Fixed} {
			t.Run(string(mode)+"/"+test.name, func(t *testing.T) {
				want := test.vulnerableStatus
				if mode == Fixed {
					want = test.fixedStatus
				}
				got := request(t, apps[mode], test.host, "alice-token", test.path)
				if got.status != want {
					t.Fatalf("status = %d, want %d; body=%q", got.status, want, got.body)
				}
				if want == 200 && !strings.Contains(got.body, `"number":"LGR-A7"`) {
					t.Errorf("200 response lacks Alice's invoice 104: %q", got.body)
				}
				if want == 404 && (strings.Contains(got.body, `"number":`) || strings.Contains(got.body, "Birch")) {
					t.Errorf("404 response exposes invoice data: %q", got.body)
				}
			})
		}
	}
}

func TestChapter09InventoryReconciliation(t *testing.T) {
	wantVulnerable := []string{
		"deprecated-still-serving:GET api.ledger.example/v1/invoices/{id}",
		"deprecated-still-serving:GET api.ledger.example/v1/invoices/{id}/pdf",
		"deprecated-still-serving:GET ledger-staging.internal/v1/invoices/{id}",
		"unlisted-host:ledger-staging.internal",
	}
	if got := ReconcileInventory(Chapter09Inventory(Vulnerable)); !reflect.DeepEqual(got, wantVulnerable) {
		t.Fatalf("vulnerable findings = %#v, want %#v", got, wantVulnerable)
	}
	if got := ReconcileInventory(Chapter09Inventory(Fixed)); len(got) != 0 {
		t.Fatalf("fixed findings = %#v, want none", got)
	}
}

func TestChapter09FixturesMatchRegisteredRoutes(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		fixture := Chapter09Inventory(mode)
		app := NewChapter9App(mode)
		if !reflect.DeepEqual(fixture.Code, app.Routes()) {
			t.Fatalf("%s code fixture = %#v, routes = %#v", mode, fixture.Code, app.Routes())
		}
	}
}

func chapter09CumulativeApp(mode Mode, clock func() time.Time) (*App, *chapter07RoundTrip) {
	rt := &chapter07RoundTrip{reply: func(*http.Request) (int, string, string) { return 200, "", "ok" }}
	resolver := chapter07Resolver(map[string][]netip.Addr{
		"logo.cedar.example": {netip.MustParseAddr("8.8.8.8")},
	}, new([]string))
	fixed := &egress{resolve: resolver,
		dial: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("canned transport should not dial")
		},
		transport: rt}
	return buildAppWithNetwork(mode, chapter09, clock,
		&chapter07Network{vulnerable: defaultFetch{&http.Client{Transport: rt}}, fixed: fixed}), rt
}

func TestChapter09CumulativeRepairs(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			now := chapter06TestTime
			app, rt := chapter09CumulativeApp(mode, func() time.Time { return now })
			get := func(path, token string) response { return chapter05Request(t, app, http.MethodGet, path, token, "") }
			if got := get("/v2/invoices/205", "alice-token"); got.status != 404 || got.body != "{\"error\":\"not found\"}\n" {
				t.Fatalf("loader and settings = %+v", got)
			}
			if got := get("/v2/invoices/104", "alice-token"); got.status != 200 || !strings.Contains(got.body, `"number":"LGR-A7"`) || strings.Contains(got.body, `"email"`) || strings.Contains(got.body, "collections_note") {
				t.Fatalf("public view = %+v", got)
			}
			if got := get("/v2/invoices/104", "dana-token"); got.status != 200 || !strings.Contains(got.body, `"email":"cedar-billing@example.test"`) || strings.Contains(got.body, "collections_note") {
				t.Fatalf("admin view = %+v", got)
			}
			if got := get("/v2/me", mobileAppKey); got.status != 401 {
				t.Fatalf("mobile key selected identity: %+v", got)
			}
			if got := chapter05Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", ""); got.status != 403 {
				t.Fatalf("role guard = %+v", got)
			}
			if got := chapter05Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", `{"reference":"PO-9","status":"paid"}`); got.status != 400 || app.store.invoices[104].Status != "open" || app.store.invoices[104].Reference == "PO-9" {
				t.Fatalf("typed patch = %+v row=%+v", got, app.store.invoices[104])
			}
			challenge := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
			for n := 1; n <= 5; n++ {
				got := chapter04Verify(t, app, challenge, "000000", "192.0.2.1")
				if n == 5 && got.status != 423 {
					t.Fatalf("OTP fifth attempt = %+v", got)
				}
			}
			if got := get("/v2/health", ""); got.status != 200 || got.body != "{\"ok\":true}\n" {
				t.Fatalf("health = %+v", got)
			}
			if got := get("/v2/admin/health", ""); got.status != 401 {
				t.Fatalf("public admin health = %+v", got)
			}
			if got := get("/v2/admin/health", "dana-token"); got.status != 200 || strings.Contains(got.body, "hosts") || strings.Contains(got.body, "debug") {
				t.Fatalf("admin health = %+v", got)
			}
			if err := validateProductionSettings(ProductionSettings(), app.Routes()); err != nil {
				t.Fatalf("settings = %v", err)
			}
			if got := chapter05Request(t, app, http.MethodPost, "/v2/webhooks/test", "alice-token", `{"url":"https://169.254.169.254/latest"}`); got.status != 400 || len(rt.requests) != 0 {
				t.Fatalf("egress guard = %+v requests=%v", got, rt.requests)
			}
			if mode == Vulnerable {
				if got := get("/v1/invoices/104/pdf", "alice-token"); got.status != 200 || strings.Contains(got.body, "cedar-billing@example.test") || len(rt.requests) != 1 {
					t.Fatalf("v1 PDF and egress = %+v requests=%v", got, rt.requests)
				}
			} else if got := get("/v1/invoices/104/pdf", "alice-token"); got.status != 404 {
				t.Fatalf("retired PDF = %+v", got)
			}
			for i := 0; i < 3; i++ {
				quoted := chapter05Request(t, app, http.MethodPost, "/v2/refunds/quote", "alice-token", `{"invoice_id":104,"amount":40000}`)
				if quoted.status != 200 {
					t.Fatalf("quote %d = %+v", i, quoted)
				}
				var q Quote
				if err := json.Unmarshal([]byte(quoted.body), &q); err != nil {
					t.Fatal(err)
				}
				confirmed := chapter05Request(t, app, http.MethodPost, "/v2/refunds/confirm", "alice-token", `{"quote_id":"`+q.ID+`"}`)
				want := 200
				if i == 2 {
					want = 202
				}
				if confirmed.status != want {
					t.Fatalf("confirm %d = %+v", i, confirmed)
				}
			}
			if app.store.invoices[104].Refunded != 80000 {
				t.Fatalf("allowance refund=%d", app.store.invoices[104].Refunded)
			}
			now = now.Add(13 * time.Hour)
			if got := get("/v2/me", "alice-token"); got.status != 401 {
				t.Fatalf("expired session = %+v", got)
			}
		})
	}
}

func TestChapter09LookupBudgetAcrossHosts(t *testing.T) {
	for _, tc := range []struct {
		mode                 Mode
		host, path, wantBody string
	}{
		{Vulnerable, StagingHost, "/v1/invoices?email=cedar-billing@example.test", "handler lookup limit"},
		{Vulnerable, PublicHost, "/v2/invoices?email=cedar-billing@example.test", "gateway lookup limit"},
		{Fixed, PublicHost, "/v2/invoices?email=cedar-billing@example.test", "gateway lookup limit"},
	} {
		app, _ := chapter09CumulativeApp(tc.mode, func() time.Time { return chapter06TestTime })
		for i := 0; i < 30; i++ {
			if got := request(t, app, tc.host, "alice-token", tc.path); got.status != 200 {
				t.Fatalf("%s %s request %d = %+v", tc.mode, tc.host, i, got)
			}
		}
		if got := request(t, app, tc.host, "alice-token", tc.path); got.status != 429 || !strings.Contains(got.body, tc.wantBody) {
			t.Fatalf("%s %s 31st = %+v", tc.mode, tc.host, got)
		}
	}
	for _, app := range []*App{NewChapter9App(Fixed), NewChapter10App(Fixed)} {
		for i := 0; i < 31; i++ {
			if got := request(t, app, PublicHost, "alice-token", "/v1/invoices?email=cedar-billing@example.test"); got.status != 404 {
				t.Fatalf("retired lookup request %d = %+v", i, got)
			}
		}
	}
}

func TestChapter09RuntimeInventoryAndRetirement(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		fixture := Chapter09Inventory(mode)
		for _, route := range fixture.Code {
			if route.Access == "" || !containsSurface(fixture.Gateway, Surface{PublicHost, route.Method, route.Pattern}) {
				t.Fatalf("%s route lacks declaration or public gateway: %+v", mode, route)
			}
			if mode == Fixed && strings.HasPrefix(route.Pattern, "/v1/") {
				t.Fatalf("fixed app still registers v1: %+v", route)
			}
		}
	}
	for _, mode := range []Mode{Vulnerable, Fixed} {
		for _, route := range NewChapter10App(mode).Routes() {
			if strings.HasPrefix(route.Pattern, "/v1/") {
				t.Fatalf("Chapter 10 %s still registers v1: %+v", mode, route)
			}
		}
	}
	if got, want := NewChapter10App(Fixed).Routes(), NewChapter9App(Fixed).Routes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Chapter 10 route table diverged from Chapter 9 fixed: got=%#v want=%#v", got, want)
	}
}
