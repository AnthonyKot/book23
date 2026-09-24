package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func chapter06Request(t *testing.T, app *App, method, path, token, body string) response {
	t.Helper()
	return chapter04Request(t, app, method, path, token, "192.0.2.66", body)
}

var chapter06TestTime = time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)

func newChapter06TestApp(mode Mode) *App {
	return buildAppWithClock(mode, chapter06, func() time.Time { return chapter06TestTime })
}

func chapter06QuoteID(t *testing.T, app *App, token string, invoiceID, amount int) string {
	t.Helper()
	got := chapter06Request(t, app, http.MethodPost, "/v2/refunds/quote", token,
		fmt.Sprintf(`{"invoice_id":%d,"amount":%d}`, invoiceID, amount))
	if got.status != 200 {
		t.Fatalf("quote invoice %d amount %d = %+v", invoiceID, amount, got)
	}
	var quote Quote
	if err := json.Unmarshal([]byte(got.body), &quote); err != nil || quote.ID == "" {
		t.Fatalf("decode quote = %+v, %v", got, err)
	}
	return quote.ID
}

func chapter06Confirm(t *testing.T, app *App, token, id string) response {
	t.Helper()
	return chapter06Request(t, app, http.MethodPost, "/v2/refunds/confirm", token, `{"quote_id":"`+id+`"}`)
}

func chapter06SeedThousand(t *testing.T, app *App) {
	t.Helper()
	for _, amount := range []int{40000, 40000, 20000} {
		id := chapter06QuoteID(t, app, "alice-token", 104, amount)
		if got := chapter06Confirm(t, app, "alice-token", id); got.status != 200 {
			t.Fatalf("seed £1,000 confirm = %+v", got)
		}
	}
	if app.store.invoices[104].Refunded != 100000 {
		t.Fatalf("seed total = %d", app.store.invoices[104].Refunded)
	}
}

func TestChapter06ExerciseSequences(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			// A: three fresh, individually valid quotes cross Cedar's daily allowance.
			app := newChapter06TestApp(mode)
			var thirdID string
			for i := 0; i < 3; i++ {
				id := chapter06QuoteID(t, app, "alice-token", 104, 40000)
				if i == 0 && id != "q-771" {
					t.Fatalf("first 104 quote = %s", id)
				}
				got := chapter06Confirm(t, app, "alice-token", id)
				if i < 2 && got.status != 200 {
					t.Fatalf("confirm %d = %+v", i, got)
				}
				thirdID = id
				if i == 2 {
					if mode == Vulnerable && got.status != 200 {
						t.Fatalf("third vulnerable confirm = %+v", got)
					}
					if mode == Fixed && (got.status != 202 || !strings.Contains(got.body, `"status":"refund_needs_approval"`)) {
						t.Fatalf("third fixed confirm = %+v", got)
					}
				}
			}
			wantAmount, wantCount, wantStatus := 120000, 3, refundConfirmed
			if mode == Fixed {
				wantAmount, wantCount, wantStatus = 80000, 2, refundNeedsApproval
			}
			if app.store.invoices[104].Refunded != wantAmount || len(app.store.refunds) != wantCount || app.store.quotes[thirdID].Status != wantStatus {
				t.Fatalf("three refunds: amount=%d count=%d quote=%+v", app.store.invoices[104].Refunded, len(app.store.refunds), app.store.quotes[thirdID])
			}
			if velocity := app.store.QuotesPerTenantHour("Cedar", chapter06TestTime); velocity != 3 {
				t.Fatalf("quote velocity = %d", velocity)
			}

			// Idempotency is safe for the same quote in either mode.
			app = newChapter06TestApp(mode)
			id := chapter06QuoteID(t, app, "alice-token", 104, 40000)
			first := chapter06Confirm(t, app, "alice-token", id)
			second := chapter06Confirm(t, app, "alice-token", id)
			if first.status != 200 || !reflect.DeepEqual(first, second) || len(app.store.refunds) != 1 || app.store.invoices[104].Refunded != 40000 {
				t.Fatalf("idempotent confirm: first=%+v second=%+v refunded=%d", first, second, app.store.invoices[104].Refunded)
			}

			// B/C: a new service identity has a fresh per-user bucket, not a fresh tenant allowance.
			app = newChapter06TestApp(mode)
			chapter06SeedThousand(t, app)
			serviceUser, ok := tenantServiceAccount(cedarKey)
			if !ok || refundedToday(usersByToken["alice-token"], app.store.refunds, chapter06TestTime) != 100000 || refundedToday(serviceUser, app.store.refunds, chapter06TestTime) != 0 {
				t.Fatal("rejected per-user counter did not expose the service-identity gap")
			}
			id = chapter06QuoteID(t, app, cedarKey, 104, 40000)
			got := chapter06Confirm(t, app, cedarKey, id)
			if mode == Vulnerable && (got.status != 200 || app.store.invoices[104].Refunded != 140000) {
				t.Fatalf("integration vulnerable confirm = %+v; refunded=%d", got, app.store.invoices[104].Refunded)
			}
			if mode == Fixed && (got.status != 202 || !strings.Contains(got.body, "refund_needs_approval") || app.store.invoices[104].Refunded != 100000) {
				t.Fatalf("integration fixed confirm = %+v; refunded=%d", got, app.store.invoices[104].Refunded)
			}

			// Dana is another identity, but approval is a distinct admin action.
			app = newChapter06TestApp(mode)
			chapter06SeedThousand(t, app)
			id = chapter06QuoteID(t, app, "dana-token", 104, 40000)
			got = chapter06Confirm(t, app, "dana-token", id)
			approvePath := "/v2/refunds/" + id + "/approve"
			alice := chapter06Request(t, app, http.MethodPost, approvePath, "alice-token", "")
			ben := chapter06Request(t, app, http.MethodPost, approvePath, "ben-token", "")
			if alice.status != 403 || ben.status != 404 {
				t.Fatalf("approval scope/role: Alice=%+v Ben=%+v", alice, ben)
			}
			if mode == Vulnerable && (got.status != 200 || app.store.invoices[104].Refunded != 140000) {
				t.Fatalf("Dana vulnerable confirm = %+v", got)
			}
			if mode == Fixed {
				if got.status != 202 || app.store.invoices[104].Refunded != 100000 {
					t.Fatalf("Dana parked confirm = %+v", got)
				}
				approved := chapter06Request(t, app, http.MethodPost, approvePath, "dana-token", "")
				if approved.status != 200 || app.store.invoices[104].Refunded != 140000 || app.store.quotes[id].Status != refundConfirmed {
					t.Fatalf("Dana approval = %+v; row=%+v", approved, app.store.invoices[104])
				}
			}

			// Chapter 1's object scope is retained at the quote step.
			app = newChapter06TestApp(mode)
			cross := chapter06Request(t, app, http.MethodPost, "/v2/refunds/quote", "ben-token", `{"invoice_id":104,"amount":40000}`)
			if cross.status != 404 || len(app.store.quotes) != 0 {
				t.Fatalf("Ben cross-tenant quote = %+v", cross)
			}

			// D decoy: quoting alone does not move money.
			app = newChapter06TestApp(mode)
			for i := 0; i < 50; i++ {
				chapter06QuoteID(t, app, "alice-token", 104, 40000)
			}
			if len(app.store.quotes) != 50 || len(app.store.refunds) != 0 || app.store.invoices[104].Refunded != 0 {
				t.Fatalf("50 quotes: quotes=%d refunds=%d", len(app.store.quotes), len(app.store.refunds))
			}
		})
	}
}

func TestChapter06ReminderExerciseAndEarlierRepairs(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := newChapter06TestApp(mode)
		path := "/v2/invoices/104/remind"
		for i := 0; i < 50; i++ {
			got := chapter06Request(t, app, http.MethodPost, path, "alice-token", "")
			if mode == Vulnerable && (got.status != 200 || !strings.Contains(got.body, "reminder_sent")) {
				t.Fatalf("%s reminder #%d = %+v", mode, i, got)
			}
			if mode == Fixed && i == 0 && (got.status != 200 || !strings.Contains(got.body, "reminder_sent")) {
				t.Fatalf("%s first reminder = %+v", mode, got)
			}
			if mode == Fixed && i > 0 && (got.status != 202 || !strings.Contains(got.body, "reminder_needs_review")) {
				t.Fatalf("%s reminder #%d = %+v", mode, i, got)
			}
		}
		count := 50
		if mode == Fixed {
			count = 1
		}
		key := fmt.Sprintf("%d:%s", 104, chapter06TestTime.UTC().Format("2006-01-02"))
		if app.store.reminders[key] != count {
			t.Fatalf("%s reminder event count = %d", mode, app.store.reminders[key])
		}
		if got := chapter06Request(t, app, http.MethodPost, "/v2/invoices/205/remind", "alice-token", ""); got.status != 404 {
			t.Fatalf("%s cross-tenant reminder = %+v", mode, got)
		}
		if got := chapter06Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", ""); got.status != 403 {
			t.Fatalf("%s Chapter 5 role regression = %+v", mode, got)
		}
		if got := chapter06Request(t, app, http.MethodGet, "/v1/invoices/205", "alice-token", ""); got.status != 404 {
			t.Fatalf("%s Chapter 1 loader regression = %+v", mode, got)
		}
		if got := chapter06Request(t, app, http.MethodGet, "/v2/invoices/104", mobileAppKey, ""); got.status != 401 {
			t.Fatalf("%s Chapter 2 identity regression = %+v", mode, got)
		}
	}
}

func TestChapter06AllowanceDayAndAtomicity(t *testing.T) {
	day := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	app := buildAppWithClock(Fixed, chapter06, func() time.Time { return day })
	ids := make([]string, 3)
	for i := range ids {
		ids[i] = chapter06QuoteID(t, app, "alice-token", 104, 40000)
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			app.store.ConfirmRefund(id, usersByToken["alice-token"], day)
		}(id)
	}
	wg.Wait()
	if len(app.store.refunds) != 2 || app.store.invoices[104].Refunded != 80000 {
		t.Fatalf("parallel confirms: refunds=%d refunded=%d", len(app.store.refunds), app.store.invoices[104].Refunded)
	}
	day = day.Add(24 * time.Hour)
	parked := ""
	for _, id := range ids {
		if app.store.quotes[id].Status == refundNeedsApproval {
			parked = id
		}
	}
	if parked == "" {
		t.Fatal("parallel confirms did not park one quote")
	}
	if _, status := app.store.ConfirmRefund(parked, usersByToken["alice-token"], day); status != refundNeedsApproval {
		t.Fatalf("parked quote left approval flow on next day: %s", status)
	}
	fresh := app.store.newQuote(usersByToken["alice-token"], app.store.invoices[104], 40000)
	if refund, status := app.store.ConfirmRefund(fresh.ID, usersByToken["alice-token"], day); status != refundConfirmed || refund.Amount != 40000 || app.store.invoices[104].Refunded != 120000 {
		t.Fatalf("next-day new quote allowance: status=%s refund=%+v", status, refund)
	}
}
