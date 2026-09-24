package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// rawInvoice is the deliberate Chapter 3 counterexample. The alias carries the
// stored fields without Invoice's MarshalJSON guard.
type rawInvoice Invoice

// Historical chapter stages retain their four-field response shape.
type legacyInvoice struct {
	ID       int    `json:"id"`
	Tenant   string `json:"tenant"`
	Amount   int    `json:"amount"`
	Refunded int    `json:"refunded"`
}

func oldInvoiceView(inv Invoice) legacyInvoice {
	return legacyInvoice{ID: inv.ID, Tenant: inv.Tenant, Amount: inv.Amount, Refunded: inv.Refunded}
}

type legacyResponseWriter struct{ http.ResponseWriter }

func legacyJSONHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(legacyResponseWriter{w}, r)
	})
}

func legacyJSONValue(value any) any {
	switch rows := value.(type) {
	case Invoice:
		return oldInvoiceView(rows)
	case []Invoice:
		out := make([]legacyInvoice, len(rows))
		for i, row := range rows {
			out[i] = oldInvoiceView(row)
		}
		return out
	default:
		return value
	}
}

type InvoiceCore struct {
	Number    string `json:"number"`
	Amount    int    `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	DueDate   string `json:"due_date"`
	Reference string `json:"reference"`
}

type CustomerNameView struct {
	Name string `json:"name"`
}

type CustomerContactView struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

type InvoiceView struct {
	InvoiceCore
	Customer CustomerNameView `json:"customer"`
}

type InvoiceAdminView struct {
	InvoiceCore
	Customer CustomerContactView `json:"customer"`
}

func invoiceCore(inv Invoice) InvoiceCore {
	return InvoiceCore{Number: inv.Number, Amount: inv.Amount, Currency: inv.Currency,
		Status: inv.Status, DueDate: inv.DueDate, Reference: inv.Reference}
}

func publicViewFor(inv Invoice) InvoiceView {
	return InvoiceView{InvoiceCore: invoiceCore(inv), Customer: CustomerNameView{Name: inv.Customer.Name}}
}

// excerpt: ch03-view
func ViewFor(user User, inv Invoice) any {
	if user.Role == "tenant-admin" || user.IsAdmin {
		return InvoiceAdminView{InvoiceCore: invoiceCore(inv), Customer: CustomerContactView{
			Name: inv.Customer.Name, Email: inv.Customer.Email,
			Phone: inv.Customer.Phone, Address: inv.Customer.Address,
		}}
	}
	return publicViewFor(inv)
}

// end excerpt

func chapter03InvoiceHandler(store *Store, mode Mode, pdf bool) http.HandlerFunc {
	if mode == Vulnerable {
		return vulnerableChapter03Read(store, pdf)
	}
	return fixedChapter03Read(store, pdf)
}

// excerpt: ch03-vulnerable-read
func vulnerableChapter03Read(store *Store, pdf bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.LoadInvoiceFor(user, id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		if pdf {
			renderRawPDF(w, invoice)
			return
		}
		inv := rawInvoice(invoice)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(inv)
	}
}

// end excerpt

func fixedChapter03Read(store *Store, pdf bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.LoadInvoiceFor(user, id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		if pdf {
			renderViewPDF(w, publicViewFor(invoice))
			return
		}
		writeJSON(w, ViewFor(user, invoice))
	}
}

func renderRawPDF(w http.ResponseWriter, invoice Invoice) {
	w.Header().Set("Content-Type", "application/pdf")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "Ledger PDF invoice=%s amount=%d customer=%s email=%s phone=%s\n",
		invoice.Number, invoice.Amount, invoice.Customer.Name, invoice.Customer.Email, invoice.Customer.Phone)
}

func renderViewPDF(w http.ResponseWriter, view InvoiceView) {
	w.Header().Set("Content-Type", "application/pdf")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "Ledger PDF invoice=%s amount=%d customer=%s\n",
		view.Number, view.Amount, view.Customer.Name)
}

func chapter03List(store *Store, mode Mode, filtered bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		invoices := store.InvoicesFor(user)
		if filtered {
			tenant := r.URL.Query().Get("tenant")
			if tenant == "" {
				http.Error(w, "tenant is required", http.StatusBadRequest)
				return
			}
			if !strings.EqualFold(tenant, user.Tenant) {
				invoices = []Invoice{}
			}
		}
		if mode == Vulnerable {
			rows := make([]rawInvoice, len(invoices))
			for i, invoice := range invoices {
				rows[i] = rawInvoice(invoice)
			}
			writeJSON(w, rows)
			return
		}
		views := make([]any, len(invoices))
		for i, invoice := range invoices {
			views[i] = ViewFor(user, invoice)
		}
		writeJSON(w, views)
	}
}

type InvoicePatch struct {
	Reference string `json:"reference"`
}

func chapter03InvoicePatch(store *Store, mode Mode) http.HandlerFunc {
	if mode == Vulnerable {
		return vulnerableChapter03Patch(store)
	}
	return fixedChapter03Patch(store)
}

// excerpt: ch03-vulnerable-patch
func vulnerableChapter03Patch(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.LoadInvoiceFor(user, id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		inv := &invoice
		if json.NewDecoder(r.Body).Decode(inv) != nil {
			http.Error(w, "invalid patch", http.StatusBadRequest)
			return
		}
		store.invoices[id] = *inv // matching exported fields came from the request
		writeJSON(w, ViewFor(user, *inv))
	}
}

// end excerpt

// excerpt: ch03-patch
func fixedChapter03Patch(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		inv, found := store.LoadInvoiceFor(user, id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		var patch InvoicePatch
		if decodeOneJSON(r, &patch) != nil {
			http.Error(w, "invalid patch field", http.StatusBadRequest)
			return
		}
		inv.Reference = patch.Reference
		store.invoices[id] = inv
		writeJSON(w, ViewFor(user, inv))
	}
}

// end excerpt

type ProfilePatch struct {
	DisplayName string `json:"display_name"`
}

func chapter03ProfilePatch(sessions *sessionStore, mode Mode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		originalName := user.Name
		if mode == Vulnerable {
			if json.NewDecoder(r.Body).Decode(&user) != nil {
				http.Error(w, "invalid profile patch", http.StatusBadRequest)
				return
			}
		} else {
			var patch ProfilePatch
			if decodeOneJSON(r, &patch) != nil {
				http.Error(w, "invalid profile field", http.StatusBadRequest)
				return
			}
			user.DisplayName = patch.DisplayName
		}
		sessions.updateUser(originalName, user)
		writeJSON(w, user)
	}
}
