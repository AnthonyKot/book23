package ledger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func chapter02Request(t *testing.T, app *App, method, path, token, claimed, body string) response {
	t.Helper()
	r := httptest.NewRequest(method, "http://"+PublicHost+path, strings.NewReader(body))
	r.Host = PublicHost
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if claimed != "" {
		r.Header.Set("X-User", claimed)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	return response{status: w.Code, body: w.Body.String()}
}

func TestChapter02ExerciseCases(t *testing.T) {
	tests := []struct {
		name, path, token, claimed string
		vulnerable, fixed          chapter01Expectation
	}{
		{"A Alice session own invoice", "/v2/invoices/104", "alice-token", "", chapter01Expectation{200, `"tenant":"Cedar"`, "Birch"}, chapter01Expectation{200, `"tenant":"Cedar"`, "Birch"}},
		{"A session ignores forged header", "/v2/me", "alice-token", "ben", chapter01Expectation{200, `"name":"Ben"`, ""}, chapter01Expectation{200, `"name":"Alice"`, "Ben"}},
		{"C Ben session impersonates Alice", "/v2/invoices/104", "ben-token", "alice", chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{404, "", "Cedar"}},
		{"v1 ignores old header with Ben session", "/v1/invoices/104", "ben-token", "alice", chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{404, "", "Cedar"}},
		{"B v1 app key impersonates Alice", "/v1/invoices/104", mobileAppKey, "alice", chapter01Expectation{200, `"tenant":"Cedar"`, ""}, chapter01Expectation{401, "", "Cedar"}},
		{"B v1 app key impersonates Ben", "/v1/invoices/205", mobileAppKey, "ben", chapter01Expectation{200, `"tenant":"Birch"`, ""}, chapter01Expectation{401, "", "Birch"}},
		{"app key alone has no identity", "/v2/me", mobileAppKey, "", chapter01Expectation{401, "", "Alice"}, chapter01Expectation{401, "", "Alice"}},
		{"D tenant key selects Birch service", "/v2/me", birchKey, "", chapter01Expectation{200, `"name":"Birch service account"`, ""}, chapter01Expectation{200, `"name":"Birch service account"`, ""}},
		{"E tenant key plus header selects Ben", "/v2/me", birchKey, "ben", chapter01Expectation{200, `"name":"Ben"`, "service account"}, chapter01Expectation{200, `"name":"Birch service account"`, `"name":"Ben"`}},
		{"tenant key reads own invoice", "/v2/invoices/205", birchKey, "ben", chapter01Expectation{200, `"tenant":"Birch"`, ""}, chapter01Expectation{200, `"tenant":"Birch"`, ""}},
		{"tenant key cannot cross tenant", "/v2/invoices/104", birchKey, "alice", chapter01Expectation{404, "", "Cedar"}, chapter01Expectation{404, "", "Cedar"}},
		{"unknown credential rejected", "/v2/me", "unknown", "alice", chapter01Expectation{401, "", "Alice"}, chapter01Expectation{401, "", "Alice"}},
	}
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter2App(mode)
		for _, test := range tests {
			t.Run(string(mode)+"/"+test.name, func(t *testing.T) {
				want := test.vulnerable
				if mode == Fixed {
					want = test.fixed
				}
				got := chapter02Request(t, app, http.MethodGet, test.path, test.token, test.claimed, "")
				if got.status != want.status || (want.include != "" && !strings.Contains(got.body, want.include)) || (want.exclude != "" && strings.Contains(got.body, want.exclude)) {
					t.Fatalf("response = %+v, want %+v", got, want)
				}
			})
		}
	}
}

func TestChapter02SessionLifetimeAndLogin(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
			app := newChapter2AppAt(mode, func() time.Time { return now })
			login := chapter02Request(t, app, http.MethodPost, "/v2/auth/login", "", "", `{"user":"Alice","password":"fixture-alice"}`)
			if login.status != 200 {
				t.Fatalf("login = %+v", login)
			}
			var issued struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal([]byte(login.body), &issued); err != nil || issued.Token == "" {
				t.Fatalf("issued token = %+v; error=%v", issued, err)
			}
			if got := chapter02Request(t, app, http.MethodGet, "/v2/invoices/104", issued.Token, "", ""); got.status != 200 {
				t.Fatalf("fresh issued session = %+v", got)
			}
			badLogin := chapter02Request(t, app, http.MethodPost, "/v2/auth/login", "", "", `{"user":"Alice","password":"wrong"}`)
			if badLogin.status != 401 || strings.Contains(badLogin.body, "token") {
				t.Fatalf("invalid credentials = %+v", badLogin)
			}
			now = now.Add(13 * time.Hour)
			want := 200
			if mode == Fixed {
				want = 401
			}
			for _, path := range []string{"/v2/invoices/104", "/v1/invoices/104", "/v2/me"} {
				got := chapter02Request(t, app, http.MethodGet, path, issued.Token, "", "")
				if got.status != want {
					t.Errorf("13-hour session %s = %+v, want %d", path, got, want)
				}
			}
		})
	}
}

func TestChapter02EarlierRepairAndRouteIdentity(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter2App(mode)
		for _, path := range []string{"/v2/invoices/205", "/v1/invoices/205", "/v1/invoices/205/pdf"} {
			got := chapter02Request(t, app, http.MethodGet, path, "alice-token", "", "")
			if got.status != 404 || strings.Contains(got.body, "Birch") {
				t.Errorf("%s %s lost Chapter 1 repair: %+v", mode, path, got)
			}
		}
		for _, path := range []string{"/v2/me/invoices", "/v2/invoices?tenant=birch"} {
			got := chapter02Request(t, app, http.MethodGet, path, "alice-token", "", "")
			if got.status != 200 || strings.Contains(got.body, `"id":205`) {
				t.Errorf("%s %s lost Chapter 1 list repair: %+v", mode, path, got)
			}
		}
		firstQuote(t, app)
		injected := postJSON(t, app, "alice-token", "/v2/refunds/confirm", `{"quote_id":"q-771","invoice_id":205}`)
		cedar, _ := app.store.Invoice(104)
		birch, _ := app.store.Invoice(205)
		if injected.status != 400 || cedar.Refunded != 0 || birch.Refunded != 0 || len(app.store.refunds) != 0 {
			t.Errorf("%s lost Chapter 1 refund repair: %+v; Cedar=%+v Birch=%+v", mode, injected, cedar, birch)
		}
	}
}

func TestChapter02CredentialBoundary(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	sessions := newSessionStore(func() time.Time { return now })
	token := sessions.issue(usersByToken["alice-token"])
	if user, ok := sessions.lookup(token, true); !ok || user.Name != "Alice" {
		t.Fatalf("fresh session = %+v, %t", user, ok)
	}
	now = now.Add(sessionLife)
	if _, ok := sessions.lookup(token, true); ok {
		t.Fatal("session valid at exact twelve-hour boundary")
	}
	if _, ok := sessions.lookup(token, false); !ok {
		t.Fatal("vulnerable counterexample did not retain expired session")
	}
	sessions.revoke(token)
	if _, ok := sessions.lookup(token, false); ok {
		t.Fatal("revoked session accepted")
	}

	app := NewChapter2App(Fixed)
	for _, route := range app.Routes() {
		if route.Pattern == "/v2/auth/login" {
			continue
		}
		path := strings.ReplaceAll(route.Pattern, "{id}", "104")
		if strings.Contains(path, "{quote}") {
			continue
		}
		got := chapter02Request(t, app, route.Method, path, mobileAppKey, "alice", "")
		if got.status != 401 {
			t.Errorf("app key reached %s %s as Alice: %+v", route.Method, route.Pattern, got)
		}
	}
}
