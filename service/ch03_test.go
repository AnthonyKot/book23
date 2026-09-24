package ledger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func chapter03JSON(t *testing.T, body string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("invalid JSON response %q: %v", body, err)
	}
	return value
}

func chapter03HasCustomerField(t *testing.T, body, field string) bool {
	t.Helper()
	value := chapter03JSON(t, body)
	customer, ok := value["customer"].(map[string]any)
	if !ok {
		t.Fatalf("missing customer object in %q", body)
	}
	_, found := customer[field]
	return found
}

func TestChapter03ExerciseReadCases(t *testing.T) {
	tests := []struct {
		name, path, token string
		wantEmail         bool
	}{
		{"A Alice item", "/v2/invoices/104", "alice-token", false},
		{"Dana admin item", "/v2/invoices/104", "dana-token", true},
		{"v1 item", "/v1/invoices/104", "alice-token", false},
	}
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter3App(mode)
		for _, test := range tests {
			t.Run(string(mode)+"/"+test.name, func(t *testing.T) {
				got := chapter02Request(t, app, http.MethodGet, test.path, test.token, "", "")
				if got.status != 200 {
					t.Fatalf("item = %+v", got)
				}
				value := chapter03JSON(t, got.body)
				if value["amount"] != float64(180000) {
					t.Errorf("amount = %v", value["amount"])
				}
				wantEmail := mode == Vulnerable || test.wantEmail
				gotEmail := chapter03HasCustomerField(t, got.body, "email")
				if gotEmail != wantEmail {
					t.Errorf("email visibility = %t, want %t; body=%q", gotEmail, wantEmail, got.body)
				}
				for _, field := range []string{"phone", "address"} {
					if visible := chapter03HasCustomerField(t, got.body, field); visible != wantEmail {
						t.Errorf("%s visibility = %t, want %t; body=%q", field, visible, wantEmail, got.body)
					}
				}
				_, margin := value["margin"]
				_, note := value["collections_note"]
				if margin != (mode == Vulnerable) || note != (mode == Vulnerable) {
					t.Errorf("internal field visibility margin=%t note=%t; body=%q", margin, note, got.body)
				}
			})
		}

		t.Run(string(mode)+"/B list decoy", func(t *testing.T) {
			got := chapter02Request(t, app, http.MethodGet, "/v2/me/invoices", "alice-token", "", "")
			if got.status != 200 {
				t.Fatalf("list = %+v", got)
			}
			var rows []map[string]any
			if err := json.Unmarshal([]byte(got.body), &rows); err != nil || len(rows) != 1 {
				t.Fatalf("list = %q; error=%v", got.body, err)
			}
			_, margin := rows[0]["margin"]
			customer, ok := rows[0]["customer"].(map[string]any)
			if !ok {
				t.Fatalf("list customer = %v", rows[0]["customer"])
			}
			_, email := customer["email"]
			if margin != (mode == Vulnerable) || email != (mode == Vulnerable) {
				t.Errorf("list property visibility margin=%t email=%t", margin, email)
			}
		})

		t.Run(string(mode)+"/C PDF", func(t *testing.T) {
			got := chapter02Request(t, app, http.MethodGet, "/v1/invoices/104/pdf", "alice-token", "", "")
			if got.status != 200 {
				t.Fatalf("PDF = %+v", got)
			}
			phone := strings.Contains(got.body, "+44 20 7946 3528")
			email := strings.Contains(got.body, "cedar-billing@example.test")
			if phone != (mode == Vulnerable) || email != (mode == Vulnerable) {
				t.Errorf("PDF contact visibility phone=%t email=%t; body=%q", phone, email, got.body)
			}
		})
	}
}

func TestChapter03ExercisePatchCases(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode)+"/D mixed reference and status", func(t *testing.T) {
			app := NewChapter3App(mode)
			got := chapter02Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", "", `{"reference":"PO-9","status":"paid"}`)
			inv, _ := app.store.Invoice(104)
			if mode == Vulnerable {
				if got.status != 200 || inv.Reference != "PO-9" || inv.Status != "paid" {
					t.Fatalf("vulnerable mixed patch = %+v; stored=%+v", got, inv)
				}
			} else if got.status != 400 || inv.Reference != "PO-Cedar" || inv.Status != "open" {
				t.Fatalf("fixed mixed patch = %+v; stored=%+v", got, inv)
			}
		})
		t.Run(string(mode)+"/D reference only", func(t *testing.T) {
			app := NewChapter3App(mode)
			got := chapter02Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", "", `{"reference":"PO-9"}`)
			inv, _ := app.store.Invoice(104)
			if got.status != 200 || inv.Reference != "PO-9" || inv.Status != "open" {
				t.Fatalf("reference-only patch = %+v; stored=%+v", got, inv)
			}
		})
		t.Run(string(mode)+"/E status and amount", func(t *testing.T) {
			app := NewChapter3App(mode)
			got := chapter02Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", "", `{"status":"paid","amount":0}`)
			inv, _ := app.store.Invoice(104)
			if mode == Vulnerable {
				if got.status != 200 || inv.Status != "paid" || inv.Amount != 0 {
					t.Fatalf("vulnerable map-equivalent patch = %+v; stored=%+v", got, inv)
				}
			} else if got.status != 400 || inv.Status != "open" || inv.Amount != 180000 {
				t.Fatalf("fixed map-equivalent patch = %+v; stored=%+v", got, inv)
			}
		})
		t.Run(string(mode)+"/E tenant reassignment", func(t *testing.T) {
			app := NewChapter3App(mode)
			got := chapter02Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", "", `{"tenant":"Birch"}`)
			inv, _ := app.store.Invoice(104)
			if mode == Vulnerable {
				if got.status != 200 || inv.Tenant != "Birch" {
					t.Fatalf("vulnerable tenant patch = %+v; stored=%+v", got, inv)
				}
			} else if got.status != 400 || inv.Tenant != "Cedar" {
				t.Fatalf("fixed tenant patch = %+v; stored=%+v", got, inv)
			}
		})
		t.Run(string(mode)+"/F profile escalation", func(t *testing.T) {
			app := NewChapter3App(mode)
			before := chapter02Request(t, app, http.MethodGet, "/v2/me", "alice-token", "", "")
			if before.status != 200 || chapter03JSON(t, before.body)["is_admin"] != false {
				t.Fatalf("initial identity = %+v", before)
			}
			got := chapter02Request(t, app, http.MethodPatch, "/v2/users/me", "alice-token", "", `{"is_admin":true}`)
			after := chapter02Request(t, app, http.MethodGet, "/v2/me", "alice-token", "", "")
			isAdmin := chapter03JSON(t, after.body)["is_admin"]
			inv, _ := app.store.Invoice(104)
			view, adminView := ViewFor(User{Name: "Alice", Tenant: "Cedar", IsAdmin: isAdmin == true}, inv).(InvoiceAdminView)
			if mode == Vulnerable {
				if got.status != 200 || isAdmin != true || !adminView || view.Customer.Email != inv.Customer.Email {
					t.Fatalf("vulnerable escalation = %+v; after=%+v; admin view=%t", got, after, adminView)
				}
			} else if got.status != 400 || isAdmin != false || adminView {
				t.Fatalf("fixed escalation = %+v; after=%+v; admin view=%t", got, after, adminView)
			}
		})
		t.Run(string(mode)+"/profile display name only", func(t *testing.T) {
			app := NewChapter3App(mode)
			got := chapter02Request(t, app, http.MethodPatch, "/v2/users/me", "alice-token", "", `{"display_name":"Alice C"}`)
			after := chapter02Request(t, app, http.MethodGet, "/v2/me", "alice-token", "", "")
			if got.status != 200 || after.status != 200 || chapter03JSON(t, after.body)["display_name"] != "Alice C" || chapter03JSON(t, after.body)["is_admin"] != false {
				t.Fatalf("valid profile patch = %+v; after=%+v", got, after)
			}
		})
		t.Run(string(mode)+"/cross-tenant patch stays denied", func(t *testing.T) {
			app := NewChapter3App(mode)
			got := chapter02Request(t, app, http.MethodPatch, "/v2/invoices/205", "alice-token", "", `{"reference":"PO-9"}`)
			inv, _ := app.store.Invoice(205)
			if got.status != 404 || inv.Reference != "PO-Birch" {
				t.Fatalf("cross-tenant patch = %+v; stored=%+v", got, inv)
			}
		})
	}
}

func TestChapter03MarshalGuardAndEarlierRepairs(t *testing.T) {
	inv, _ := seedStore().Invoice(104)
	if _, err := json.Marshal(inv); err == nil || !strings.Contains(err.Error(), "encode a view, not the row") {
		t.Fatalf("raw invoice encoded: %v", err)
	}
	if _, err := json.Marshal([]Invoice{inv}); err == nil {
		t.Fatal("raw invoice slice encoded")
	}
	w := httptest.NewRecorder()
	writeJSON(w, inv)
	if w.Code != 500 || strings.Contains(w.Body.String(), "margin") {
		t.Fatalf("unguarded raw response = %d %q", w.Code, w.Body.String())
	}
	for name, old := range map[string]*App{
		"Chapter 1":       NewApp(Fixed),
		"Chapter 2":       NewChapter2App(Fixed),
		"Chapter 9 pilot": NewChapter9App(Vulnerable),
	} {
		got := request(t, old, PublicHost, "alice-token", "/v2/invoices/104")
		if got.status != 200 || strings.Contains(got.body, "customer") || strings.Contains(got.body, "margin") {
			t.Errorf("%s historical response changed: %+v", name, got)
		}
	}

	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter3App(mode)
		for _, path := range []string{"/v2/invoices/205", "/v1/invoices/205", "/v1/invoices/205/pdf"} {
			got := chapter02Request(t, app, http.MethodGet, path, "alice-token", "", "")
			if got.status != 404 || strings.Contains(got.body, "Birch") {
				t.Errorf("%s lost Chapter 1 tenant check on %s: %+v", mode, path, got)
			}
		}
		for _, path := range []string{"/v2/invoices/104", "/v1/invoices/104", "/v1/invoices/104/pdf"} {
			got := chapter02Request(t, app, http.MethodGet, path, "ben-token", "alice", "")
			if got.status != 404 {
				t.Errorf("%s lost Chapter 2 identity repair on %s: %+v", mode, path, got)
			}
		}
		if got := chapter02Request(t, app, http.MethodGet, "/v1/invoices/104", mobileAppKey, "alice", ""); got.status != 401 {
			t.Errorf("%s accepts shared mobile key: %+v", mode, got)
		}
		firstQuote(t, app)
		injected := postJSON(t, app, "alice-token", "/v2/refunds/confirm", `{"quote_id":"q-771","invoice_id":205}`)
		if injected.status != 400 || len(app.store.refunds) != 0 {
			t.Errorf("%s lost Chapter 1 refund repair: %+v", mode, injected)
		}
	}
}

func TestChapter03SessionExpiryAndAllReadViews(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
		app := buildAppWithClock(mode, chapter03, func() time.Time { return now })
		for _, route := range app.Routes() {
			if route.Method != http.MethodGet || !strings.Contains(route.Pattern, "invoices") {
				continue
			}
			path := strings.ReplaceAll(route.Pattern, "{id}", "104")
			if route.Pattern == "/v2/invoices" {
				path += "?tenant=cedar"
			}
			got := chapter02Request(t, app, http.MethodGet, path, "alice-token", "", "")
			if got.status != 200 {
				t.Fatalf("%s %s = %+v", mode, route.Pattern, got)
			}
			if mode == Fixed && (strings.Contains(got.body, "cedar-billing@example.test") || strings.Contains(got.body, "collections_note") || strings.Contains(got.body, "margin")) {
				t.Errorf("fixed %s leaks internal properties: %q", route.Pattern, got.body)
			}
		}
		now = now.Add(13 * time.Hour)
		if got := chapter02Request(t, app, http.MethodGet, "/v2/invoices/104", "alice-token", "", ""); got.status != 401 {
			t.Errorf("%s accepts expired session: %+v", mode, got)
		}
	}
}
