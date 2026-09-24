package ledger

import "sort"

type Mode string

const (
	Vulnerable Mode = "vulnerable"
	Fixed      Mode = "fixed"
)

type User struct {
	Name   string `json:"name"`
	Tenant string `json:"tenant"`
	Role   string `json:"role"`
}

type Invoice struct {
	ID       int    `json:"id"`
	Tenant   string `json:"tenant"`
	Amount   int    `json:"amount"`
	Refunded int    `json:"refunded"`
}

type Store struct {
	invoices        map[int]Invoice
	quotes          map[string]Quote
	refunds         []Refund
	nextQuoteNumber int
}

func seedStore() *Store {
	return &Store{invoices: map[int]Invoice{
		104: {ID: 104, Tenant: "Cedar", Amount: 180000}, // pence: £1,800.00
		205: {ID: 205, Tenant: "Birch", Amount: 64000},  // pence: £640.00
	}, quotes: make(map[string]Quote), nextQuoteNumber: 772}
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
