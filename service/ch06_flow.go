package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	refundConfirmed     = "confirmed"
	refundNeedsApproval = "needs_approval"
	refundMissing       = "missing"
	refundInvalid       = "invalid"
)

func chapter06Quote(store *Store, clock func() time.Time) http.HandlerFunc {
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
		if decodeOneJSON(r, &input) != nil {
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
		store.noteQuote(user.Tenant, clock())
		writeJSON(w, quote)
	}
}

func chapter06ConfirmInput(w http.ResponseWriter, r *http.Request, store *Store) (User, Quote, bool) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return User{}, Quote{}, false
	}
	var input struct {
		QuoteID   string          `json:"quote_id"`
		InvoiceID json.RawMessage `json:"invoice_id"`
	}
	if decodeOneJSON(r, &input) != nil || input.QuoteID == "" || len(input.InvoiceID) != 0 {
		http.Error(w, "invalid confirm request", http.StatusBadRequest)
		return User{}, Quote{}, false
	}
	quote, found := store.QuoteFor(user, input.QuoteID)
	if !found {
		http.NotFound(w, r)
		return User{}, Quote{}, false
	}
	return user, quote, true
}

func writeChapter06Decision(w http.ResponseWriter, status string, refund Refund) {
	switch status {
	case refundConfirmed:
		writeJSON(w, refund)
	case refundNeedsApproval:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, struct {
			Status string `json:"status"`
		}{"refund_needs_approval"})
	default:
		http.Error(w, "invalid refund target or balance", http.StatusBadRequest)
	}
}

// excerpt: ch06-vulnerable-confirm
func vulnerableChapter06Confirm(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, quote, ok := chapter06ConfirmInput(w, r, store)
		if !ok {
			return // session, body and quote tenant were checked
		}
		refund, status := store.UnrestrictedRefund(quote.ID, user, clock())
		writeChapter06Decision(w, status, refund)
	}
}

func (s *Store) UnrestrictedRefund(id string, actor User, at time.Time) (Refund, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quote, ok := s.quotes[id]
	if !ok || quote.Tenant != actor.Tenant {
		return Refund{}, refundMissing
	}
	if quote.Status == refundConfirmed {
		refund, found := s.refundByQuoteLocked(id)
		if found {
			return refund, refundConfirmed
		}
		return Refund{}, refundInvalid
	}
	if quote.Status == refundNeedsApproval {
		return Refund{}, refundNeedsApproval
	}
	invoice, found := s.invoices[quote.InvoiceID]
	if !found || quote.Amount <= 0 || quote.Amount > invoice.Amount-invoice.Refunded {
		return Refund{}, refundInvalid
	}
	return s.recordRefundLocked(quote, actor, at), refundConfirmed
}

// end excerpt

// excerpt: ch06-fixed-confirm
func fixedChapter06Confirm(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, quote, ok := chapter06ConfirmInput(w, r, store)
		if !ok {
			return
		}
		refund, status := store.ConfirmRefund(quote.ID, user, clock())
		writeChapter06Decision(w, status, refund)
	}
}

// end excerpt

func (s *Store) QuoteFor(user User, id string) (Quote, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quote, ok := s.quotes[id]
	return quote, ok && quote.Tenant == user.Tenant
}

func (s *Store) refundByQuoteLocked(id string) (Refund, bool) {
	for _, refund := range s.refunds {
		if refund.QuoteID == id {
			return refund, true
		}
	}
	return Refund{}, false
}

func (s *Store) recordRefundLocked(quote Quote, actor User, at time.Time) Refund {
	invoice := s.invoices[quote.InvoiceID]
	invoice.Refunded += quote.Amount
	s.invoices[quote.InvoiceID] = invoice
	refund := Refund{QuoteID: quote.ID, InvoiceID: quote.InvoiceID, Tenant: quote.Tenant,
		Amount: quote.Amount, At: at, Actor: actor.Name}
	s.refunds = append(s.refunds, refund)
	quote.Status = refundConfirmed
	s.quotes[quote.ID] = quote
	return refund
}

// excerpt: ch06-tenant-allowance
func (s *Store) ConfirmRefund(id string, actor User, at time.Time) (Refund, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quote, ok := s.quotes[id]
	if !ok || quote.Tenant != actor.Tenant {
		return Refund{}, refundMissing
	}
	if quote.Status == refundConfirmed {
		refund, found := s.refundByQuoteLocked(id)
		if found {
			return refund, refundConfirmed
		}
		return Refund{}, refundInvalid
	}
	if quote.Status == refundNeedsApproval {
		return Refund{}, refundNeedsApproval
	}
	invoice, found := s.invoices[quote.InvoiceID]
	if !found || quote.Amount <= 0 || quote.Amount > invoice.Amount-invoice.Refunded {
		return Refund{}, refundInvalid
	}
	today := at.UTC().Format("2006-01-02")
	total := 0
	for _, refund := range s.refunds {
		if refund.Tenant == quote.Tenant && refund.At.UTC().Format("2006-01-02") == today {
			total += refund.Amount
		}
	}
	if total+quote.Amount > s.allowances[quote.Tenant] {
		quote.Status = refundNeedsApproval
		s.quotes[id] = quote
		return Refund{}, refundNeedsApproval
	}
	return s.recordRefundLocked(quote, actor, at), refundConfirmed
}

// end excerpt

// excerpt: ch06-per-user-counter
func refundedToday(user User, refunds []Refund, at time.Time) int {
	total := 0
	today := at.UTC().Format("2006-01-02")
	for _, refund := range refunds {
		if refund.Actor == user.Name && refund.At.UTC().Format("2006-01-02") == today {
			total += refund.Amount
		}
	}
	return total
}

// end excerpt

func (s *Store) ApproveRefund(id string, actor User, at time.Time) (Refund, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quote, ok := s.quotes[id]
	if !ok || quote.Tenant != actor.Tenant {
		return Refund{}, refundMissing
	}
	if quote.Status != refundNeedsApproval {
		return Refund{}, refundInvalid
	}
	invoice, found := s.invoices[quote.InvoiceID]
	if !found || quote.Amount <= 0 || quote.Amount > invoice.Amount-invoice.Refunded {
		return Refund{}, refundInvalid
	}
	return s.recordRefundLocked(quote, actor, at), refundConfirmed
}

func chapter06Approve(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := currentUser(r)
		refund, status := store.ApproveRefund(r.PathValue("quote"), user, clock())
		writeChapter06Decision(w, status, refund)
	}
}

func (s *Store) noteQuote(tenant string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quoteVelocity[fmt.Sprintf("%s:%s", tenant, at.UTC().Format("2006-01-02T15"))]++
}

func (s *Store) QuotesPerTenantHour(tenant string, at time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.quoteVelocity[fmt.Sprintf("%s:%s", tenant, at.UTC().Format("2006-01-02T15"))]
}

func chapter06Remind(store *Store, mode Mode, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoice := r.Context().Value(scopedInvoiceKey{}).(Invoice)
		key := fmt.Sprintf("%d:%s", invoice.ID, clock().UTC().Format("2006-01-02"))
		store.mu.Lock()
		count := store.reminders[key]
		if mode == Fixed && count >= 1 {
			store.mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
			writeJSON(w, struct {
				Status string `json:"status"`
			}{"reminder_needs_review"})
			return
		}
		store.reminders[key] = count + 1 // fixture event; no email is sent
		store.mu.Unlock()
		writeJSON(w, struct {
			Status string `json:"status"`
		}{"reminder_sent"})
	}
}

func registerChapter06Routes(mux *http.ServeMux, routes []Route, store *Store, mode Mode, clock func() time.Time) []Route {
	added := []Route{
		{Method: http.MethodPost, Pattern: "/v2/refunds/{quote}/approve"},
		{Method: http.MethodPost, Pattern: "/v2/invoices/{id}/remind"},
	}
	mux.HandleFunc("POST /v2/refunds/{quote}/approve", chapter06Approve(store, clock))
	mux.HandleFunc("POST /v2/invoices/{id}/remind", chapter06Remind(store, mode, clock))
	return declareRoutes(append(routes, added...), append(chapter05Declarations, chapter06Declarations...))
}

func scopeQuote(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		parts := splitPath(r.URL.Path)
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		if _, found := store.QuoteFor(user, parts[len(parts)-2]); !found {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func splitPath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}
