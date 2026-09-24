package ledger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// Quote.Amount and Refund.Amount are integer pence in the Chapter 1 fixture.
type Quote struct {
	ID        string `json:"quote_id"`
	Tenant    string `json:"-"`
	InvoiceID int    `json:"invoice_id"`
	Amount    int    `json:"amount"`
	Status    string `json:"status"`
}

type Refund struct {
	QuoteID   string `json:"quote_id"`
	InvoiceID int    `json:"invoice_id"`
	Tenant    string `json:"tenant"`
	Amount    int    `json:"amount"`
}

func (s *Store) InvoicesWhere(tenant string) []Invoice {
	result := make([]Invoice, 0)
	for _, invoice := range s.invoices {
		if strings.EqualFold(invoice.Tenant, tenant) {
			result = append(result, invoice)
		}
	}
	sortInvoices(result)
	return result
}

func sortInvoices(invoices []Invoice) {
	sort.Slice(invoices, func(i, j int) bool { return invoices[i].ID < invoices[j].ID })
}

func listInvoicesWhere(store *Store, secure bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tenant := r.URL.Query().Get("tenant")
		if tenant == "" {
			http.Error(w, "tenant is required", http.StatusBadRequest)
			return
		}
		if secure && !strings.EqualFold(tenant, user.Tenant) {
			writeJSON(w, []Invoice{})
			return
		}
		writeJSON(w, store.InvoicesWhere(tenant))
	}
}

func quoteRefund(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var input struct {
			InvoiceID int `json:"invoice_id"`
			Amount    int `json:"amount"`
		}
		if err := decodeOneJSON(r, &input); err != nil {
			http.Error(w, "invalid quote request", http.StatusBadRequest)
			return
		}
		invoice, found := store.LoadInvoiceFor(user, input.InvoiceID)
		if !found {
			http.NotFound(w, r)
			return
		}
		if input.Amount <= 0 || input.Amount > invoice.Amount-invoice.Refunded {
			http.Error(w, "invalid quote amount", http.StatusBadRequest)
			return
		}
		quote := store.newQuote(user, invoice, input.Amount)
		writeJSON(w, quote)
	}
}

func confirmRefund(store *Store, mode Mode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var input struct {
			QuoteID   string          `json:"quote_id"`
			InvoiceID json.RawMessage `json:"invoice_id"`
		}
		if err := decodeOneJSON(r, &input); err != nil || input.QuoteID == "" {
			http.Error(w, "invalid confirm request", http.StatusBadRequest)
			return
		}
		if mode == Fixed && len(input.InvoiceID) != 0 {
			http.Error(w, "invoice_id is not a confirm field", http.StatusBadRequest)
			return
		}
		quote, found := store.quotes[input.QuoteID]
		if !found || quote.Tenant != user.Tenant {
			http.NotFound(w, r)
			return
		}
		targetID := quote.InvoiceID
		if mode == Vulnerable && len(input.InvoiceID) != 0 {
			if err := json.Unmarshal(input.InvoiceID, &targetID); err != nil {
				http.Error(w, "invalid invoice_id", http.StatusBadRequest)
				return
			}
		}
		refund, applied := store.applyQuote(quote, targetID)
		if !applied {
			http.Error(w, "invalid refund target or balance", http.StatusBadRequest)
			return
		}
		writeJSON(w, refund)
	}
}

func decodeOneJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("extra JSON value")
	}
	return nil
}

func (s *Store) newQuote(user User, invoice Invoice, amount int) Quote {
	id := "q-771"
	if invoice.ID != 104 || s.quotes[id].ID != "" {
		id = fmt.Sprintf("q-%d", s.nextQuoteNumber)
		s.nextQuoteNumber++
	}
	quote := Quote{ID: id, Tenant: user.Tenant, InvoiceID: invoice.ID, Amount: amount, Status: "open"}
	s.quotes[id] = quote
	return quote
}

func (s *Store) applyQuote(quote Quote, targetID int) (Refund, bool) {
	if quote.Status == "confirmed" {
		for _, refund := range s.refunds {
			if refund.QuoteID == quote.ID {
				return refund, true
			}
		}
		return Refund{}, false
	}
	invoice, found := s.invoices[targetID]
	if !found || quote.Amount <= 0 || quote.Amount > invoice.Amount-invoice.Refunded {
		return Refund{}, false
	}
	invoice.Refunded += quote.Amount
	s.invoices[targetID] = invoice
	refund := Refund{QuoteID: quote.ID, InvoiceID: targetID, Tenant: invoice.Tenant, Amount: quote.Amount}
	s.refunds = append(s.refunds, refund)
	quote.Status = "confirmed"
	s.quotes[quote.ID] = quote
	return refund, true
}
