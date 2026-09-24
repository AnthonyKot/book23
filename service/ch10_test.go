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

func chapter10Fixture(mode Mode, rt *chapter07RoundTrip) *App {
	fixed := &egress{
		resolve: chapter07Resolver(map[string][]netip.Addr{"rates.partner.example": {netip.MustParseAddr("8.8.8.8")}}, new([]string)),
		dial: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("canned transport should not dial")
		},
		transport: rt,
	}
	return buildAppWithNetwork(mode, chapter10, func() time.Time { return chapter06TestTime },
		&chapter07Network{vulnerable: defaultFetch{&http.Client{Transport: rt}}, fixed: fixed})
}

func chapter10Reply(status int, location, body string) *chapter07RoundTrip {
	return &chapter07RoundTrip{reply: func(r *http.Request) (int, string, string) {
		if r.URL.Path == "/v2/latest" {
			return 200, "", `{"EUR":0.95}`
		}
		return status, location, body
	}}
}

func chapter10Poll(t *testing.T, app *App, rt *chapter07RoundTrip) error {
	t.Helper()
	before := http.DefaultTransport
	http.DefaultTransport = rt // serial fixture only; http.Get retains normal redirect handling
	defer func() { http.DefaultTransport = before }()
	return app.PollRates(context.Background())
}

func TestChapter10RateExerciseCases(t *testing.T) {
	cases := []struct {
		name, body, location                 string
		status, vulnerableEUR, vulnerableGBP int
		fixedRejected                        bool
		vulnerableRate, fixedRate            float64
		vulnerableRequests, fixedRequests    int
	}{
		{"decoy_92", `{"EUR":92}`, "", 200, 2760000, 180000, true, 92, 0.92, 1, 1},
		{"baseline_092", `{"EUR":0.92}`, "", 200, 27600, 180000, false, 0.92, 0.92, 1, 1},
		{"negative", `{"EUR":-0.92}`, "", 200, -27600, 180000, true, -0.92, 0.92, 1, 1},
		{"extra_GBP", `{"EUR":0.92,"GBP":0.5}`, "", 200, 27600, 90000, true, 0.92, 0.92, 1, 1},
		{"redirect", "", "https://rates.partner.example/v2/latest", 302, 28500, 180000, true, 0.95, 0.92, 2, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []Mode{Vulnerable, Fixed} {
				t.Run(string(mode), func(t *testing.T) {
					// Each response is fed to both modes. Rejections keep the last-good row.
					rt := chapter10Reply(tc.status, tc.location, tc.body)
					app := chapter10Fixture(mode, rt)
					before := app.store.lastGoodRate
					err := chapter10Poll(t, app, rt)
					wantRequests := tc.vulnerableRequests
					if mode == Fixed {
						wantRequests = tc.fixedRequests
					}
					if len(rt.requests) != wantRequests {
						t.Fatalf("trace=%v; want %d", rt.requests, wantRequests)
					}
					if mode == Fixed && tc.fixedRejected && err == nil {
						t.Fatal("invalid partner row accepted")
					}
					if mode == Fixed && !tc.fixedRejected && err != nil {
						t.Fatal(err)
					}
					if mode == Vulnerable && err != nil {
						t.Fatal(err)
					}
					if mode == Fixed && tc.fixedRejected && (app.store.lastGoodRate != before || app.store.opsPages != 1) {
						t.Fatalf("rejected row changed state: rate=%+v ops=%d", app.store.lastGoodRate, app.store.opsPages)
					}
					if mode == Fixed && !tc.fixedRejected && (app.store.lastGoodRate.AsOf != chapter06TestTime || app.store.opsPages != 0) {
						t.Fatalf("accepted row has bad metadata: rate=%+v ops=%d", app.store.lastGoodRate, app.store.opsPages)
					}
					euro := app.store.CreateInvoice(Invoice{ID: 412, Tenant: "Cedar", Currency: "EUR", AmountEUR: 300}, mode)
					pounds := app.store.CreateInvoice(Invoice{ID: 104, Tenant: "Cedar", Currency: "GBP", Amount: 180000}, mode)
					wantEUR, wantGBP, wantRate := tc.vulnerableEUR, tc.vulnerableGBP, tc.vulnerableRate
					if mode == Fixed {
						wantEUR, wantGBP, wantRate = 27600, 180000, tc.fixedRate
					}
					if euro.Amount != wantEUR || pounds.Amount != wantGBP {
						t.Fatalf("invoices EUR=%d GBP=%d, want %d/%d", euro.Amount, pounds.Amount, wantEUR, wantGBP)
					}
					if mode == Vulnerable {
						if app.store.rates["EUR"] != wantRate || euro.FXRate != 0 {
							t.Fatalf("vulnerable rate=%v invoice=%+v", app.store.rates, euro)
						}
					} else if app.store.lastGoodRate.Value != wantRate || euro.FXRate != wantRate || euro.AmountEUR != 300 {
						t.Fatalf("fixed rate=%+v invoice=%+v", app.store.lastGoodRate, euro)
					}
					if tc.name == "extra_GBP" && mode == Fixed && len(app.store.rates) != 1 {
						t.Fatalf("fixed rate table changed: %v", app.store.rates)
					}
				})
			}
		})
	}
}

func TestChapter10RecordedRateRefund(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			rt := chapter10Reply(200, "", `{"EUR":0.95}`)
			app := chapter10Fixture(mode, rt)
			if app.store.invoices[412].Amount != 27600 {
				t.Fatal("EUR invoice must precede changed rate")
			}
			if err := chapter10Poll(t, app, rt); err != nil {
				t.Fatal(err)
			}
			readsBefore := app.store.rateReads
			quote := chapter05Request(t, app, http.MethodPost, "/v2/refunds/quote", "alice-token", `{"invoice_id":412,"amount":300}`)
			if mode == Vulnerable {
				if quote.status != 400 || len(app.store.quotes) != 0 {
					t.Fatalf("vulnerable quote=%+v quotes=%v", quote, app.store.quotes)
				}
				return
			}
			if quote.status != 200 || !strings.Contains(quote.body, `"amount":27600`) || app.store.rateReads != readsBefore {
				t.Fatalf("fixed quote=%+v reads=%d/%d", quote, app.store.rateReads, readsBefore)
			}
			var q Quote
			for _, stored := range app.store.quotes {
				q = stored
			}
			if q.InvoiceID != 412 || q.Amount != 27600 {
				t.Fatalf("stored quote=%+v", q)
			}
			confirmed := chapter05Request(t, app, http.MethodPost, "/v2/refunds/confirm", "alice-token", `{"quote_id":"`+q.ID+`"}`)
			if confirmed.status != 200 || !strings.Contains(confirmed.body, `"amount":27600`) || app.store.invoices[412].Refunded != 27600 || len(app.store.refunds) != 1 || app.store.refunds[0].Amount != 27600 || app.store.rateReads != readsBefore {
				t.Fatalf("confirm=%+v invoice=%+v refunds=%v reads=%d", confirmed, app.store.invoices[412], app.store.refunds, app.store.rateReads)
			}
			// The quote amount is pence before Chapter 6's tenant allowance decision.
			if app.store.refunds[0].Tenant != "Cedar" || app.store.refunds[0].At.UTC().Format("2006-01-02") != chapter06TestTime.UTC().Format("2006-01-02") {
				t.Fatalf("allowance record=%+v", app.store.refunds[0])
			}
			// 27,600 + 80,000 pence exceeds Cedar's 100,000-pence UTC-day allowance.
			larger := chapter05Request(t, app, http.MethodPost, "/v2/refunds/quote", "alice-token", `{"invoice_id":104,"amount":80000}`)
			if larger.status != 200 {
				t.Fatalf("second quote=%+v", larger)
			}
			var second Quote
			for _, stored := range app.store.quotes {
				if stored.InvoiceID == 104 {
					second = stored
				}
			}
			parked := chapter05Request(t, app, http.MethodPost, "/v2/refunds/confirm", "alice-token", `{"quote_id":"`+second.ID+`"}`)
			if parked.status != 202 || len(app.store.refunds) != 1 || app.store.quotes[second.ID].Status != refundNeedsApproval {
				t.Fatalf("allowance: result=%+v refunds=%v quote=%+v", parked, app.store.refunds, app.store.quotes[second.ID])
			}
			legacy := app.store.invoices[412]
			legacy.FXRate, legacy.Refunded = 0, 0
			app.store.invoices[412] = legacy
			beforeOps := app.store.opsPages
			legacyQuote := chapter05Request(t, app, http.MethodPost, "/v2/refunds/quote", "alice-token", `{"invoice_id":412,"amount":300}`)
			if legacyQuote.status != 409 || !strings.Contains(legacyQuote.body, "rate_reconciliation_required") || app.store.opsPages != beforeOps+1 || app.store.rateReads != readsBefore {
				t.Fatalf("legacy=%+v ops=%d reads=%d", legacyQuote, app.store.opsPages, app.store.rateReads)
			}
		})
	}
}

func TestChapter10EarlierRepairsAndRetirement(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := chapter10Fixture(mode, chapter10Reply(200, "", `{"EUR":0.92}`))
		if got := chapter05Request(t, app, http.MethodGet, "/v1/invoices/104", "alice-token", ""); got.status != 404 {
			t.Fatalf("v1 retirement %s=%+v", mode, got)
		}
		staging := httptest.NewRequest(http.MethodGet, "http://"+StagingHost+"/v2/invoices/104", nil)
		staging.Host = StagingHost
		staging.Header.Set("Authorization", "Bearer alice-token")
		stagingReply := httptest.NewRecorder()
		app.ServeHTTP(stagingReply, staging)
		if stagingReply.Code != 404 {
			t.Fatalf("staging retirement %s=%d", mode, stagingReply.Code)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/205", "alice-token", ""); got.status != 404 || got.body != "{\"error\":\"not found\"}\n" {
			t.Fatalf("loader/settings %s=%+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", ""); got.status != 403 {
			t.Fatalf("role %s=%+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/104", "alice-token", ""); got.status != 200 || strings.Contains(got.body, "collections_note") {
			t.Fatalf("view %s=%+v", mode, got)
		}
	}
}
