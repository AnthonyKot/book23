package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const partnerRateURL = "https://rates.partner.example/latest"

type Rate struct {
	Value float64
	AsOf  time.Time
}

func seedChapter10(store *Store, mode Mode, at time.Time) {
	store.rates = map[string]float64{"EUR": 0.92}
	store.lastGoodRate = Rate{Value: 0.92, AsOf: at.Add(-time.Hour)}
	store.CreateInvoice(Invoice{ID: 412, Tenant: "Cedar", Number: "LGR-E4", Currency: "EUR",
		AmountEUR: 300, Status: "open", Customer: Customer{Name: "Cedar Euro Customer"}}, mode)
}

func (s *Store) CreateInvoice(inv Invoice, mode Mode) Invoice {
	if mode == Vulnerable {
		return s.vulnerableCreateInvoice(inv)
	}
	return s.fixedCreateInvoice(inv)
}

// excerpt: ch10-rates-vulnerable
func (s *Store) vulnerablePollRates() error {
	reply, err := http.Get(partnerRateURL) // default client follows redirects
	if err != nil {
		return err
	}
	defer reply.Body.Close()
	if reply.StatusCode != http.StatusOK {
		return fmt.Errorf("partner status %d", reply.StatusCode)
	}
	var row map[string]float64
	if err := json.NewDecoder(reply.Body).Decode(&row); err != nil {
		return err
	}
	s.rates = row // every key in the partner response is trusted
	return nil
}

func (s *Store) vulnerableCreateInvoice(inv Invoice) Invoice {
	if rate, ok := s.rates[inv.Currency]; ok {
		amount := float64(inv.Amount)
		if inv.Currency == "EUR" {
			amount = inv.AmountEUR * 100
		}
		inv.Amount = int(math.Round(amount * rate))
	}
	s.invoices[inv.ID] = inv
	return inv
}

// end excerpt

// excerpt: ch10-rate-row
type rateRow struct {
	EUR float64 `json:"EUR"`
}

// end excerpt

// excerpt: ch10-rates-fixed
func decodeRateRow(body io.Reader) (rateRow, error) {
	var row rateRow
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&row); err != nil {
		return rateRow{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return rateRow{}, fmt.Errorf("extra partner JSON")
	}
	if row.EUR < 0.70 || row.EUR > 1.10 || math.IsNaN(row.EUR) || math.IsInf(row.EUR, 0) {
		return rateRow{}, fmt.Errorf("EUR rate outside accepted band")
	}
	return row, nil
}

func (s *Store) fixedPollRates(ctx context.Context, fetcher outboundFetcher, at time.Time) error {
	reply, err := fetcher.Fetch(ctx, partnerRateURL) // shared egress pins DNS and refuses redirects
	if err != nil {
		s.opsPages++
		return err
	}
	defer reply.Body.Close()
	if reply.StatusCode != http.StatusOK {
		s.opsPages++
		return fmt.Errorf("partner status %d", reply.StatusCode)
	}
	row, err := decodeRateRow(reply.Body)
	if err != nil {
		s.opsPages++
		return err
	}
	s.lastGoodRate = Rate{Value: row.EUR, AsOf: at}
	return nil
}

func (s *Store) fixedCreateInvoice(inv Invoice) Invoice {
	if inv.Currency == "EUR" {
		s.rateReads++ // only invoice creation reads the current partner rate
		inv.FXRate = s.lastGoodRate.Value
		inv.Amount = int(math.Round(inv.AmountEUR * inv.FXRate * 100))
	}
	s.invoices[inv.ID] = inv // GBP never consults the feed
	return inv
}

// end excerpt
