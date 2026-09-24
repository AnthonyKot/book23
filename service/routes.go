package ledger

import (
	"fmt"
	"net/http"
)

type Access string

const (
	Public      Access = "Public"
	UserAccess  Access = "User"
	TenantAdmin Access = "TenantAdmin"
)

// excerpt: ch05-access-decl
var chapter05Declarations = []Route{
	{Method: http.MethodGet, Pattern: "/v2/invoices/{id}", Access: UserAccess},
	{Method: http.MethodPatch, Pattern: "/v2/invoices/{id}", Access: UserAccess},
	{Method: http.MethodDelete, Pattern: "/v2/invoices/{id}", Access: TenantAdmin},
	{Method: http.MethodGet, Pattern: "/v2/admin/users", Access: TenantAdmin},
	{Method: http.MethodPatch, Pattern: "/v2/admin/users/{id}", Access: TenantAdmin},
	{Method: http.MethodPost, Pattern: "/v2/admin/invoices/{id}/void", Access: TenantAdmin},
	{Method: http.MethodGet, Pattern: "/v1/invoices/{id}", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v1/invoices/{id}/pdf", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v2/me/invoices", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v2/invoices", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/refunds/quote", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/refunds/confirm", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v1/invoices", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/auth/login", Access: Public},
	{Method: http.MethodGet, Pattern: "/v2/me", Access: UserAccess},
	{Method: http.MethodPatch, Pattern: "/v2/users/me", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/auth/otp/request", Access: Public},
	{Method: http.MethodPost, Pattern: "/v2/auth/otp/verify", Access: Public},
}

// end excerpt

func declareChapter05Routes(routes []Route) []Route {
	return declareRoutes(routes, chapter05Declarations)
}

var chapter06Declarations = []Route{
	{Method: http.MethodPost, Pattern: "/v2/refunds/{quote}/approve", Access: TenantAdmin},
	{Method: http.MethodPost, Pattern: "/v2/invoices/{id}/remind", Access: UserAccess},
}

var chapter07Declarations = []Route{
	{Method: http.MethodPost, Pattern: "/v2/webhooks/test", Access: UserAccess},
}

func declareRoutes(routes, policy []Route) []Route {
	declarations := make(map[string]Access, len(policy))
	for _, route := range policy {
		key := route.Method + " " + route.Pattern
		if route.Access != Public && route.Access != UserAccess && route.Access != TenantAdmin {
			panic("ledger: invalid access declaration: " + key)
		}
		if _, exists := declarations[key]; exists {
			panic("ledger: duplicate access declaration: " + key)
		}
		declarations[key] = route.Access
	}
	declared := make([]Route, len(routes))
	for i, route := range routes {
		key := route.Method + " " + route.Pattern
		access, ok := declarations[key]
		if !ok || route.Access != "" && route.Access != access {
			panic(fmt.Sprintf("ledger: missing or conflicting access declaration: %s", key))
		}
		route.Access = access
		declared[i] = route
		delete(declarations, key)
	}
	if len(declarations) != 0 {
		panic("ledger: access declaration has no registered route")
	}
	return declared
}
