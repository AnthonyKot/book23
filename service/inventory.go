package ledger

import (
	"fmt"
	"sort"
)

const StagingHost = "ledger-staging.internal"

type Surface struct {
	Host   string
	Method string
	Route  string
}

func (s Surface) String() string {
	return fmt.Sprintf("%s %s%s", s.Method, s.Host, s.Route)
}

type InventoryFixture struct {
	DeclaredHosts []string
	Code          []Route
	Gateway       []Surface
	Traffic       []Surface
	Deprecated    []Surface
}

// excerpt: ch09-code-route-fixture
var vulnerableCodeRoutes = []Route{
	{Method: "GET", Pattern: "/v2/invoices/{id}"},
	{Method: "GET", Pattern: "/v2/me/invoices"},
	{Method: "GET", Pattern: "/v1/invoices/{id}"},
	{Method: "GET", Pattern: "/v1/invoices/{id}/pdf"},
}

var fixedCodeRoutes = vulnerableCodeRoutes[:2]

// end excerpt

// excerpt: ch09-deprecated-surfaces
var vulnerableDeprecated = []Surface{
	{PublicHost, "GET", "/v1/invoices/{id}"},
	{PublicHost, "GET", "/v1/invoices/{id}/pdf"},
	{StagingHost, "GET", "/v1/invoices/{id}"},
}

// end excerpt

// excerpt: ch09-vulnerable-inventory-fixture
var vulnerableInventory = InventoryFixture{
	DeclaredHosts: []string{PublicHost},
	Code:          vulnerableCodeRoutes,
	Gateway: []Surface{
		{PublicHost, "GET", "/v2/invoices/{id}"},
		{PublicHost, "GET", "/v2/me/invoices"},
		{PublicHost, "GET", "/v1/invoices/{id}"},
		{PublicHost, "GET", "/v1/invoices/{id}/pdf"},
		{StagingHost, "GET", "/v1/invoices/{id}"},
	},
	Traffic: []Surface{
		{PublicHost, "GET", "/v2/invoices/{id}"},
		{PublicHost, "GET", "/v2/me/invoices"},
		{PublicHost, "GET", "/v1/invoices/{id}"},
		{PublicHost, "GET", "/v1/invoices/{id}/pdf"},
		{StagingHost, "GET", "/v1/invoices/{id}"},
	},
	Deprecated: vulnerableDeprecated,
}

// end excerpt

var fixedInventory = InventoryFixture{
	DeclaredHosts: []string{PublicHost},
	Code:          fixedCodeRoutes,
	Gateway: []Surface{
		{PublicHost, "GET", "/v2/invoices/{id}"},
		{PublicHost, "GET", "/v2/me/invoices"},
	},
	Traffic: []Surface{
		{PublicHost, "GET", "/v2/invoices/{id}"},
		{PublicHost, "GET", "/v2/me/invoices"},
	},
}

func Chapter09Inventory(mode Mode) InventoryFixture {
	fixture := vulnerableInventory
	if mode == Fixed {
		fixture = fixedInventory
	}
	// The printed four-route fixture isolates the chapter's before/after pair.
	// Reconciliation uses the complete registered table, including earlier repairs.
	fixture.Code = NewChapter9App(mode).Routes()
	fixture.Gateway = append([]Surface(nil), fixture.Gateway...)
	for _, route := range fixture.Code {
		public := Surface{PublicHost, route.Method, route.Pattern}
		if !containsSurface(fixture.Gateway, public) {
			fixture.Gateway = append(fixture.Gateway, public)
		}
		if mode == Vulnerable && len(route.Pattern) >= 4 && route.Pattern[:4] == "/v1/" {
			staging := Surface{StagingHost, route.Method, route.Pattern}
			if !containsSurface(fixture.Gateway, staging) {
				fixture.Gateway = append(fixture.Gateway, staging)
			}
		}
	}
	return fixture
}

// excerpt: ch09-reconcile-inventory
func ReconcileInventory(f InventoryFixture) []string {
	findings := make([]string, 0)
	findings = append(findings, unlistedHostFindings(f)...)
	findings = append(findings, deprecatedSurfaceFindings(f)...)
	findings = append(findings, configurationFindings(f)...)
	sort.Strings(findings)
	return findings
}

// end excerpt

func unlistedHostFindings(f InventoryFixture) []string {
	findings := make([]string, 0)
	for _, host := range observedHosts(f.Gateway, f.Traffic) {
		if !contains(f.DeclaredHosts, host) {
			findings = append(findings, "unlisted-host:"+host)
		}
	}
	return findings
}

func deprecatedSurfaceFindings(f InventoryFixture) []string {
	findings := make([]string, 0)
	for _, surface := range f.Deprecated {
		if containsSurface(f.Gateway, surface) || containsSurface(f.Traffic, surface) {
			findings = append(findings, "deprecated-still-serving:"+surface.String())
		}
	}
	return findings
}

func configurationFindings(f InventoryFixture) []string {
	findings := make([]string, 0)
	for _, surface := range f.Gateway {
		if !containsRoute(f.Code, surface.Method, surface.Route) {
			findings = append(findings, "gateway-route-without-code:"+surface.String())
		}
	}
	for _, surface := range f.Traffic {
		if !containsSurface(f.Gateway, surface) {
			findings = append(findings, "traffic-without-gateway:"+surface.String())
		}
	}
	for _, host := range f.DeclaredHosts {
		for _, route := range f.Code {
			surface := Surface{Host: host, Method: route.Method, Route: route.Pattern}
			if !containsSurface(f.Gateway, surface) {
				findings = append(findings, "code-route-without-gateway:"+surface.String())
			}
		}
	}
	return findings
}

func observedHosts(groups ...[]Surface) []string {
	set := make(map[string]bool)
	for _, group := range groups {
		for _, surface := range group {
			set[surface.Host] = true
		}
	}
	result := make([]string, 0, len(set))
	for host := range set {
		result = append(result, host)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsSurface(values []Surface, wanted Surface) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsRoute(values []Route, method, pattern string) bool {
	for _, value := range values {
		if value.Method == method && value.Pattern == pattern {
			return true
		}
	}
	return false
}
