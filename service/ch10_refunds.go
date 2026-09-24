package ledger

import (
	"math"
	"net/http"
	"time"
)

type chapter10QuoteInput struct {
	InvoiceID int     `json:"invoice_id"`
	Amount    float64 `json:"amount"`
}

func chapter10QuoteRequest(w http.ResponseWriter, r *http.Request, store *Store) (User, Invoice, chapter10QuoteInput, bool) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return User{}, Invoice{}, chapter10QuoteInput{}, false
	}
	var input chapter10QuoteInput
	if decodeOneJSON(r, &input) != nil || input.Amount <= 0 || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) {
		http.Error(w, "invalid quote request", http.StatusBadRequest)
		return User{}, Invoice{}, chapter10QuoteInput{}, false
	}
	invoice, found := store.LoadInvoiceFor(user, input.InvoiceID)
	if !found {
		http.NotFound(w, r)
		return User{}, Invoice{}, chapter10QuoteInput{}, false
	}
	if invoice.Currency != "EUR" && math.Trunc(input.Amount) != input.Amount {
		http.Error(w, "invalid quote amount", http.StatusBadRequest)
		return User{}, Invoice{}, chapter10QuoteInput{}, false
	}
	return user, invoice, input, true
}

func chapter10Pence(input float64, rate float64) int { return int(math.Round(input * rate * 100)) }

// excerpt: ch10-refund-quote-vulnerable
func vulnerableChapter10Quote(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, inv, input, ok := chapter10QuoteRequest(w, r, store) // calls LoadInvoiceFor
		if !ok {
			return
		}
		amountPence := int(input.Amount) // GBP requests already carry integer pence
		if inv.Currency == "EUR" {
			amountPence = chapter10Pence(input.Amount, store.rates["EUR"])
		}
		if amountPence <= 0 || amountPence > inv.Amount-inv.Refunded {
			http.Error(w, "invalid quote amount", http.StatusBadRequest)
			return
		}
		quote := store.newQuote(user, inv, amountPence)
		store.noteQuote(user.Tenant, clock())
		writeJSON(w, quote)
	}
}

// end excerpt

// excerpt: ch10-refund-quote-fixed
func fixedChapter10Quote(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, inv, input, ok := chapter10QuoteRequest(w, r, store) // calls LoadInvoiceFor
		if !ok {
			return
		}
		amountPence := int(input.Amount) // GBP requests already carry integer pence
		if inv.Currency == "EUR" {
			if inv.FXRate == 0 {
				store.opsPages++ // manual reconciliation, never today's rate
				http.Error(w, "rate_reconciliation_required", http.StatusConflict)
				return
			}
			amountPence = chapter10Pence(input.Amount, inv.FXRate)
		}
		if amountPence <= 0 || amountPence > inv.Amount-inv.Refunded {
			http.Error(w, "invalid quote amount", http.StatusBadRequest)
			return
		}
		quote := store.newQuote(user, inv, amountPence)
		store.noteQuote(user.Tenant, clock())
		writeJSON(w, quote)
	}
}

// end excerpt
