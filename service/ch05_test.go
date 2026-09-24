package ledger

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func chapter05Request(t *testing.T, app *App, method, path, token, body string) response {
	t.Helper()
	return chapter04Request(t, app, method, path, token, "192.0.2.44", body)
}

func TestChapter05ExerciseCases(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			// A: PATCH is a permitted user operation, although it writes the row.
			app := NewChapter5App(mode)
			got := chapter05Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", `{"reference":"PO-9"}`)
			if got.status != 200 || !strings.Contains(got.body, `"reference":"PO-9"`) || app.store.invoices[104].Reference != "PO-9" {
				t.Fatalf("A PATCH = %+v; row=%+v", got, app.store.invoices[104])
			}

			// B and B': same URL and verb, different role.
			app = NewChapter5App(mode)
			got = chapter05Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", "")
			if mode == Vulnerable && (got.status != 204 || hasInvoice(app.store, 104)) {
				t.Fatalf("B vulnerable DELETE = %+v; row present=%v", got, hasInvoice(app.store, 104))
			}
			if mode == Fixed && (got.status != 403 || !hasInvoice(app.store, 104)) {
				t.Fatalf("B fixed DELETE = %+v; row present=%v", got, hasInvoice(app.store, 104))
			}
			app = NewChapter5App(mode)
			got = chapter05Request(t, app, http.MethodDelete, "/v2/invoices/104", "dana-token", "")
			if got.status != 204 || hasInvoice(app.store, 104) {
				t.Fatalf("B' Dana DELETE = %+v; row present=%v", got, hasInvoice(app.store, 104))
			}

			// C: tenant scope rejects a cross-tenant row before the admin role matters.
			app = NewChapter5App(mode)
			got = chapter05Request(t, app, http.MethodDelete, "/v2/invoices/205", "dana-token", "")
			if got.status != 404 || !hasInvoice(app.store, 205) {
				t.Fatalf("C cross-tenant DELETE = %+v", got)
			}

			// D and D': user list is tenant-scoped but still needs a role gate.
			got = chapter05Request(t, app, http.MethodGet, "/v2/admin/users", "alice-token", "")
			if mode == Vulnerable && (got.status != 200 || !strings.Contains(got.body, `"name":"Dana"`) || strings.Contains(got.body, `"name":"Ben"`)) {
				t.Fatalf("D vulnerable list = %+v", got)
			}
			if mode == Fixed && got.status != 403 {
				t.Fatalf("D fixed list = %+v", got)
			}
			got = chapter05Request(t, app, http.MethodGet, "/v2/admin/users", "dana-token", "")
			if got.status != 200 || !strings.Contains(got.body, `"name":"Alice"`) || strings.Contains(got.body, `"name":"Ben"`) {
				t.Fatalf("D' Dana list = %+v", got)
			}

			// E: the parallel admin edit reopens the Chapter 3 self-promotion path.
			app = NewChapter5App(mode)
			got = chapter05Request(t, app, http.MethodPatch, "/v2/admin/users/alice", "alice-token", `{"role":"tenant-admin"}`)
			view := chapter05Request(t, app, http.MethodGet, "/v2/invoices/104", "alice-token", "")
			if mode == Vulnerable && (got.status != 200 || !strings.Contains(got.body, `"role":"tenant-admin"`) || !strings.Contains(view.body, `"email":"cedar-billing@example.test"`)) {
				t.Fatalf("E vulnerable promotion = %+v; view=%+v", got, view)
			}
			if mode == Fixed && (got.status != 403 || strings.Contains(view.body, `"email"`)) {
				t.Fatalf("E fixed promotion = %+v; view=%+v", got, view)
			}

			// E': the typed self-service patch remains closed in both modes.
			app = NewChapter5App(mode)
			got = chapter05Request(t, app, http.MethodPatch, "/v2/users/me", "alice-token", `{"role":"tenant-admin"}`)
			me := chapter05Request(t, app, http.MethodGet, "/v2/me", "alice-token", "")
			if got.status != 400 || !strings.Contains(me.body, `"role":"user"`) {
				t.Fatalf("E' self patch = %+v; me=%+v", got, me)
			}

			// F: Go's pattern match does not silently grant a case variant.
			got = chapter05Request(t, app, http.MethodGet, "/v2/Admin/users", "alice-token", "")
			if got.status != 404 {
				t.Fatalf("F case-variant path = %+v", got)
			}
		})
	}
}

func hasInvoice(store *Store, id int) bool {
	_, ok := store.Invoice(id)
	return ok
}

func TestChapter05AdminVoidAndScopeOrder(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter5App(mode)
		wrongRole := chapter05Request(t, app, http.MethodPost, "/v2/admin/invoices/104/void", "alice-token", "")
		want := 204
		if mode == Fixed {
			want = 403
		}
		if wrongRole.status != want || (app.store.invoices[104].Status == "void") != (mode == Vulnerable) {
			t.Fatalf("%s void wrong role = %+v; status=%s", mode, wrongRole, app.store.invoices[104].Status)
		}
		crossTenant := chapter05Request(t, app, http.MethodPost, "/v2/admin/invoices/205/void", "alice-token", "")
		if crossTenant.status != 404 || app.store.invoices[205].Status != "open" {
			t.Fatalf("%s void cross tenant = %+v", mode, crossTenant)
		}
		crossUser := chapter05Request(t, app, http.MethodPatch, "/v2/admin/users/ben", "alice-token", `{"role":"tenant-admin"}`)
		if crossUser.status != 404 {
			t.Fatalf("%s admin user cross tenant = %+v", mode, crossUser)
		}
		app = NewChapter5App(mode)
		admin := chapter05Request(t, app, http.MethodPost, "/v2/admin/invoices/104/void", "dana-token", "")
		if admin.status != 204 || app.store.invoices[104].Status != "void" {
			t.Fatalf("%s Dana void = %+v", mode, admin)
		}
		promote := chapter05Request(t, app, http.MethodPatch, "/v2/admin/users/alice", "dana-token", `{"role":"tenant-admin"}`)
		if promote.status != 200 || !strings.Contains(promote.body, `"role":"tenant-admin"`) {
			t.Fatalf("%s Dana user edit = %+v", mode, promote)
		}
	}
}

// excerpt: ch05-declares-test
func TestEveryRouteDeclares(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		for _, route := range NewChapter5App(mode).Routes() {
			if route.Access != Public && route.Access != UserAccess && route.Access != TenantAdmin {
				t.Fatalf("%s %s %s has no access level", mode, route.Method, route.Pattern)
			}
		}
	}
}

// end excerpt

func TestChapter05MissingDeclarationFailsRegistration(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("route without access declaration registered")
		}
	}()
	declareChapter05Routes(append(NewChapter5App(Fixed).Routes(), Route{Method: http.MethodGet, Pattern: "/forgotten"}))
}

func TestChapter05EarlierRepairs(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter5App(mode)
		if got := chapter05Request(t, app, http.MethodGet, "/v1/invoices/205", "alice-token", ""); got.status != 404 {
			t.Fatalf("%s v1 loader = %+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v1/invoices/104/pdf", "alice-token", ""); got.status != 200 || strings.Contains(got.body, "email") {
			t.Fatalf("%s v1 PDF view = %+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/104", mobileAppKey, ""); got.status != 401 {
			t.Fatalf("%s mobile key identity = %+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices?limit=60", "alice-token", ""); got.status != 200 {
			t.Fatalf("%s page route = %+v", mode, got)
		}
		challenge := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
		for i := 0; i < 4; i++ {
			if got := chapter04Verify(t, app, challenge, "000000", "192.0.2.1"); got.status != 400 {
				t.Fatalf("%s wrong OTP #%d = %+v", mode, i+1, got)
			}
		}
		if got := chapter04Verify(t, app, challenge, "000000", "192.0.2.2"); got.status != 423 {
			t.Fatalf("%s fifth OTP = %+v", mode, got)
		}
		// Fixed Chapter 2 expiry remains in force for both Chapter 5 modes.
		now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
		clocked := buildAppWithClock(mode, chapter05, func() time.Time { return now })
		now = now.Add(sessionLife)
		if got := chapter05Request(t, clocked, http.MethodGet, "/v2/me", "alice-token", ""); got.status != 401 {
			t.Fatalf("%s expired session = %+v", mode, got)
		}
	}
}
