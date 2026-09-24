package ledger

import (
	"errors"
	"sort"
	"sync"
)

type Mode string

const (
	Vulnerable Mode = "vulnerable"
	Fixed      Mode = "fixed"
)

type User struct {
	Name        string `json:"name"`
	Tenant      string `json:"tenant"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name,omitempty"`
	IsAdmin     bool   `json:"is_admin"`
}

type Customer struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

type Invoice struct {
	ID              int      `json:"id"`
	Tenant          string   `json:"tenant"`
	Number          string   `json:"number"`
	Amount          int      `json:"amount"`
	Refunded        int      `json:"refunded"`
	LineItems       []string `json:"line_items"`
	Currency        string   `json:"currency"`
	Status          string   `json:"status"`
	DueDate         string   `json:"due_date"`
	Reference       string   `json:"reference"`
	Customer        Customer `json:"customer"`
	CollectionsNote string   `json:"collections_note"`
	Margin          int      `json:"margin"`
}

// excerpt: ch03-marshal-guard
func (Invoice) MarshalJSON() ([]byte, error) {
	return nil, errors.New("encode a view, not the row")
}

// end excerpt

type Store struct {
	mu              sync.Mutex
	invoices        map[int]Invoice
	quotes          map[string]Quote
	refunds         []Refund
	allowances      map[string]int
	quoteVelocity   map[string]int
	reminders       map[string]int
	nextQuoteNumber int
}

func seedStore() *Store {
	return &Store{invoices: map[int]Invoice{
		104: {ID: 104, Tenant: "Cedar", Number: "LGR-A7", Amount: 180000, LineItems: []string{"Cedar billing"},
			Currency: "GBP", Status: "open", DueDate: "2026-10-31", Reference: "PO-Cedar",
			Customer:        Customer{Name: "Cedar Customer", Email: "cedar-billing@example.test", Phone: "+44 20 7946 3528", Address: "Cedar Lane"},
			CollectionsNote: "Ledger-only follow-up", Margin: 36000}, // pence: £1,800.00
		205: {ID: 205, Tenant: "Birch", Number: "LGR-K8", Amount: 64000, LineItems: []string{"Birch billing"},
			Currency: "GBP", Status: "open", DueDate: "2026-11-18", Reference: "PO-Birch",
			Customer:        Customer{Name: "Birch Customer", Email: "birch-billing@example.test", Phone: "+44 20 7946 8613", Address: "Birch Road"},
			CollectionsNote: "Ledger-only review", Margin: 12800}, // pence: £640.00
	}, quotes: make(map[string]Quote), allowances: map[string]int{"Cedar": 100000, "Birch": 100000},
		quoteVelocity: make(map[string]int), reminders: make(map[string]int), nextQuoteNumber: 772}
}

func (s *Store) Invoice(id int) (Invoice, bool) {
	invoice, ok := s.invoices[id]
	return invoice, ok
}

// excerpt: ch01-load-invoice-for
func (s *Store) LoadInvoiceFor(user User, id int) (Invoice, bool) {
	invoice, ok := s.invoices[id]
	if !ok || invoice.Tenant != user.Tenant {
		return Invoice{}, false
	}
	return invoice, true
}

// end excerpt

func (s *Store) InvoicesFor(user User) []Invoice {
	result := make([]Invoice, 0)
	for _, invoice := range s.invoices {
		if invoice.Tenant == user.Tenant {
			result = append(result, invoice)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

var usersByToken = map[string]User{
	"alice-token": {Name: "Alice", Tenant: "Cedar", Role: "user"},
	"ben-token":   {Name: "Ben", Tenant: "Birch", Role: "user"},
	"dana-token":  {Name: "Dana", Tenant: "Cedar", Role: "tenant-admin"},
}
