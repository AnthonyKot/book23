package ledger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func chapter04Request(t *testing.T, app http.Handler, method, path, token, ip, body string) response {
	t.Helper()
	r := httptest.NewRequest(method, "http://"+PublicHost+path, strings.NewReader(body))
	r.Host = PublicHost
	r.RemoteAddr = ip + ":42424"
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	return response{status: w.Code, body: w.Body.String()}
}

func chapter04Challenge(t *testing.T, app *App, account string) response {
	t.Helper()
	return chapter04Request(t, app, http.MethodPost, "/v2/auth/otp/request", "", "192.0.2.1", `{"account_id":"`+account+`"}`)
}

func chapter04ChallengeID(t *testing.T, result response) string {
	t.Helper()
	if result.status != 200 {
		t.Fatalf("challenge request = %+v", result)
	}
	var body struct {
		ChallengeID string `json:"challenge_id"`
	}
	if err := json.Unmarshal([]byte(result.body), &body); err != nil || body.ChallengeID == "" {
		t.Fatalf("invalid challenge response = %+v; error=%v", result, err)
	}
	return body.ChallengeID
}

func chapter04Verify(t *testing.T, app *App, id, code, ip string) response {
	t.Helper()
	return chapter04Request(t, app, http.MethodPost, "/v2/auth/otp/verify", "", ip,
		`{"account_id":"Ben","challenge_id":"`+id+`","code":"`+code+`"}`)
}

func TestChapter04OTPExerciseCases(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		for _, rotating := range []bool{false, true} {
			name := "A same IP"
			if rotating {
				name = "A prime rotating IPs"
			}
			t.Run(string(mode)+"/"+name, func(t *testing.T) {
				now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
				app := buildAppWithClock(mode, chapter04, func() time.Time { return now })
				id := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
				for n := 1; n <= 6; n++ {
					ip := "192.0.2.1"
					if rotating {
						ip = "192.0.2." + strconv.Itoa(n)
					}
					got := chapter04Verify(t, app, id, "000000", ip)
					want := 400
					if mode == Fixed && n >= otpTargetBudget {
						want = 423
					}
					if got.status != want || (want == 423 && !strings.Contains(got.body, "locked")) {
						t.Fatalf("wrong try %d from %s = %+v; want %d", n, ip, got, want)
					}
				}
				if mode == Fixed {
					if got := chapter04Verify(t, app, id, fixtureOTPCode, "192.0.2.200"); got.status != 423 {
						t.Fatalf("locked challenge accepted correct code: %+v", got)
					}
				}
			})
		}
		t.Run(string(mode)+"/A double prime locked reissue", func(t *testing.T) {
			now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
			app := buildAppWithClock(mode, chapter04, func() time.Time { return now })
			id := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
			for n := 1; n <= 5; n++ {
				got := chapter04Verify(t, app, id, "000000", "192.0.2.1")
				if mode == Fixed && n == 5 && got.status != 423 {
					t.Fatalf("fifth try = %+v", got)
				}
			}
			reissue := chapter04Challenge(t, app, "Ben")
			if mode == Vulnerable {
				newID := chapter04ChallengeID(t, reissue)
				if newID == id || chapter04Verify(t, app, newID, "000000", "192.0.2.2").status != 400 {
					t.Fatalf("vulnerable reissue did not reset budget: %+v", reissue)
				}
			} else if reissue.status != 423 || !strings.Contains(reissue.body, "locked") {
				t.Fatalf("fixed reissue = %+v", reissue)
			}
			now = now.Add(otpWindow)
			newID := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
			if newID == id || chapter04Verify(t, app, newID, "000000", "192.0.2.2").status != 400 {
				t.Fatalf("fresh window did not issue usable challenge: %s", newID)
			}
		})
		if mode == Fixed {
			t.Run("fixed/active reissue retains attempts", func(t *testing.T) {
				app := NewChapter4App(Fixed)
				id := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
				for n := 0; n < 4; n++ {
					if got := chapter04Verify(t, app, id, "000000", "192.0.2.1"); got.status != 400 {
						t.Fatalf("wrong attempt %d = %+v", n+1, got)
					}
				}
				if reissued := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben")); reissued != id {
					t.Fatalf("active challenge replaced from %s to %s", id, reissued)
				}
				if fifth := chapter04Verify(t, app, id, "000000", "192.0.2.2"); fifth.status != 423 {
					t.Fatalf("reissue reset target attempts: %+v", fifth)
				}
			})
		}
		t.Run(string(mode)+"/correct code spent and session bound to Ben", func(t *testing.T) {
			app := NewChapter4App(mode)
			id := chapter04ChallengeID(t, chapter04Challenge(t, app, "Ben"))
			got := chapter04Verify(t, app, id, fixtureOTPCode, "192.0.2.1")
			var issued struct {
				Token string `json:"token"`
			}
			if got.status != 200 || json.Unmarshal([]byte(got.body), &issued) != nil || issued.Token == "" {
				t.Fatalf("correct code = %+v", got)
			}
			if replay := chapter04Verify(t, app, id, fixtureOTPCode, "192.0.2.2"); replay.status != 400 {
				t.Fatalf("used code replay = %+v", replay)
			}
			me := chapter04Request(t, app, http.MethodGet, "/v2/me", issued.Token, "192.0.2.1", "")
			if me.status != 200 || !strings.Contains(me.body, `"name":"Ben"`) {
				t.Fatalf("OTP session identity = %+v", me)
			}
		})
	}
	clock := func() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) }
	guard := perIPGuard(newWindowLimiter(clock, lookupWindow), otpIPBudget, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for n := 1; n <= otpIPBudget+1; n++ {
		got := chapter04Request(t, guard, http.MethodPost, "/v2/auth/otp/verify", "", "192.0.2.1", "")
		want := 200
		if n > otpIPBudget {
			want = 429
		}
		if got.status != want {
			t.Fatalf("per-IP guard request %d = %+v; want %d", n, got, want)
		}
	}
	if got := chapter04Request(t, guard, http.MethodPost, "/v2/auth/otp/verify", "", "192.0.2.2", ""); got.status != 200 {
		t.Fatalf("separate IP did not get own guard budget: %+v", got)
	}
}

func TestChapter04LookupExerciseCases(t *testing.T) {
	for _, route := range []string{"/v2/invoices", "/v1/invoices"} {
		for _, mode := range []Mode{Vulnerable, Fixed} {
			t.Run(string(mode)+route, func(t *testing.T) {
				now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
				app := buildAppWithClock(mode, chapter04, func() time.Time { return now })
				path := route + "?email=cedar-billing@example.test"
				for n := 1; n <= 31; n++ {
					got := chapter04Request(t, app, http.MethodGet, path, "alice-token", "192.0.2.1", "")
					want := 200
					if n == 31 && (mode == Fixed || route == "/v2/invoices") {
						want = 429
					}
					if got.status != want {
						t.Fatalf("same-IP lookup %d = %+v; want %d", n, got, want)
					}
					if want == 200 && (strings.Contains(got.body, "cedar-billing@example.test") || strings.Contains(got.body, "margin")) {
						t.Fatalf("lookup lost Chapter 3 view: %+v", got)
					}
					if want == 200 && (!strings.Contains(got.body, "LGR-A7") || strings.Contains(got.body, "LGR-K8")) {
						t.Fatalf("lookup lost tenant scope or match: %+v", got)
					}
				}
				now = now.Add(lookupWindow)
				for n := 1; n <= 60; n++ {
					ip := "198.51.100." + strconv.Itoa(n)
					got := chapter04Request(t, app, http.MethodGet, path, "alice-token", ip, "")
					want := 200
					if mode == Fixed && n > 30 {
						want = 429
					}
					if got.status != want {
						t.Fatalf("rotating-IP lookup %d = %+v; want %d", n, got, want)
					}
				}
			})
		}
	}
}

func TestChapter04PageExerciseCases(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			app := NewChapter4App(mode)
			base, _ := app.store.Invoice(104)
			for n := 0; n < 59; n++ {
				row := base
				row.ID = 1000 + n // test-only rows; canonical invoices remain 104 and 205
				app.store.invoices[row.ID] = row
			}
			countRows := func(path string) int {
				t.Helper()
				got := chapter04Request(t, app, http.MethodGet, path, "alice-token", "192.0.2.1", "")
				if got.status != 200 {
					t.Fatalf("page = %+v", got)
				}
				var rows []map[string]any
				if err := json.Unmarshal([]byte(got.body), &rows); err != nil {
					t.Fatal(err)
				}
				return len(rows)
			}
			want := 60
			if mode == Fixed {
				want = maximumPageSize
			}
			if got := countRows("/v2/invoices?limit=1000000"); got != want {
				t.Fatalf("large page has %d rows, want %d", got, want)
			}
			for n := 0; n < 10000; n++ {
				got := chapter04Request(t, app, http.MethodGet, "/v2/invoices?limit=50", "alice-token", "192.0.2.1", "")
				if got.status != 200 {
					t.Fatalf("fifty-row request %d = %+v", n+1, got)
				}
				if n == 9999 {
					var rows []map[string]any
					if err := json.Unmarshal([]byte(got.body), &rows); err != nil || len(rows) != 50 {
						t.Fatalf("last page length = %d, error=%v", len(rows), err)
					}
				}
			}
		})
	}
}

func TestChapter04SharedHandlerAndEarlierRepairs(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	limits := newChapter04Limits(clock)
	handler := chapter04Lookup(seedStore(), Fixed, limits, "v1")
	for n := 1; n <= 31; n++ {
		r := httptest.NewRequest(http.MethodGet, "http://"+PublicHost+"/v1/invoices?email=cedar-billing@example.test", nil)
		r = r.WithContext(context.WithValue(r.Context(), principalContextKey{}, requestPrincipal{usersByToken["alice-token"], true}))
		w := httptest.NewRecorder()
		handler(w, r)
		want := 200
		if n == 31 {
			want = 429
		}
		if w.Code != want || (n == 31 && !strings.Contains(w.Body.String(), "handler lookup limit")) {
			t.Fatalf("shared handler lookup %d = %d %q; want %d", n, w.Code, w.Body.String(), want)
		}
	}
	called := 0
	gateway := chapter04Gateway(Fixed, newChapter04Limits(clock), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	}))
	for n := 1; n <= 31; n++ {
		r := httptest.NewRequest(http.MethodGet, "http://"+PublicHost+"/v1/invoices?email=cedar-billing@example.test", nil)
		r = r.WithContext(context.WithValue(r.Context(), principalContextKey{}, requestPrincipal{usersByToken["alice-token"], true}))
		w := httptest.NewRecorder()
		gateway.ServeHTTP(w, r)
		want := 200
		if n == 31 {
			want = 429
		}
		if w.Code != want || (n == 31 && !strings.Contains(w.Body.String(), "gateway lookup limit")) {
			t.Fatalf("gateway lookup %d = %d %q; want %d", n, w.Code, w.Body.String(), want)
		}
	}
	if called != 30 {
		t.Fatalf("gateway passed %d requests downstream, want 30", called)
	}
	separateKeys := buildAppWithClock(Fixed, chapter04, clock)
	for n := 0; n < 30; n++ {
		got := chapter04Request(t, separateKeys, http.MethodGet, "/v2/invoices?email=cedar-billing@example.test", "alice-token", "192.0.2.1", "")
		if got.status != 200 {
			t.Fatalf("v2 route budget ended at %d: %+v", n, got)
		}
	}
	if byID := chapter04Request(t, separateKeys, http.MethodGet, "/v2/invoices/104", "alice-token", "192.0.2.1", ""); byID.status != 200 {
		t.Errorf("email lookup budget incorrectly blocks invoice-by-ID: %+v", byID)
	}
	for _, test := range []struct{ path, token string }{
		{"/v1/invoices?email=cedar-billing@example.test", "alice-token"},
		{"/v2/invoices?email=birch-billing@example.test", "ben-token"},
	} {
		got := chapter04Request(t, separateKeys, http.MethodGet, test.path, test.token, "192.0.2.1", "")
		if got.status != 200 {
			t.Errorf("tenant/route budgets were merged: %+v", got)
		}
	}

	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter4App(mode)
		for _, path := range []string{"/v2/invoices/205", "/v1/invoices/205", "/v1/invoices/205/pdf"} {
			got := chapter04Request(t, app, http.MethodGet, path, "alice-token", "192.0.2.1", "")
			if got.status != 404 {
				t.Errorf("%s lost Chapter 1 loader on %s: %+v", mode, path, got)
			}
		}
		if got := chapter04Request(t, app, http.MethodGet, "/v1/invoices/104", mobileAppKey, "192.0.2.1", ""); got.status != 401 {
			t.Errorf("%s accepts shared mobile key: %+v", mode, got)
		}
		if got := chapter04Request(t, app, http.MethodGet, "/v2/invoices/104", "ben-token", "192.0.2.1", ""); got.status != 404 {
			t.Errorf("%s lost Chapter 2 identity repair: %+v", mode, got)
		}
		if got := chapter04Request(t, app, http.MethodGet, "/v2/invoices/104", "alice-token", "192.0.2.1", ""); got.status != 200 || strings.Contains(got.body, "margin") {
			t.Errorf("%s lost Chapter 3 view: %+v", mode, got)
		}
		patch := chapter04Request(t, app, http.MethodPatch, "/v2/invoices/104", "alice-token", "192.0.2.1", `{"status":"paid"}`)
		inv, _ := app.store.Invoice(104)
		if patch.status != 400 || inv.Status != "open" {
			t.Errorf("%s lost Chapter 3 typed patch: %+v; inv=%+v", mode, patch, inv)
		}
		firstQuote(t, app)
		injected := postJSON(t, app, "alice-token", "/v2/refunds/confirm", `{"quote_id":"q-771","invoice_id":205}`)
		if injected.status != 400 || len(app.store.refunds) != 0 {
			t.Errorf("%s lost Chapter 1 refund repair: %+v", mode, injected)
		}
	}
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter4App(mode)
		otherTenant := chapter04Request(t, app, http.MethodGet, "/v2/invoices?email=birch-billing@example.test", "alice-token", "192.0.2.1", "")
		if otherTenant.status != 200 || strings.TrimSpace(otherTenant.body) != "[]" {
			t.Errorf("%s email lookup crossed tenant boundary: %+v", mode, otherTenant)
		}
		filtered := chapter04Request(t, app, http.MethodGet, "/v2/invoices?tenant=birch", "alice-token", "192.0.2.1", "")
		if filtered.status != 200 || strings.TrimSpace(filtered.body) != "[]" {
			t.Errorf("%s tenant filter lost Chapter 1 repair: %+v", mode, filtered)
		}
	}
}
