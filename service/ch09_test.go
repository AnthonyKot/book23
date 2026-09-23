package ledger

import (
	"reflect"
	"testing"
)

func TestChapter09ExerciseCases(t *testing.T) {
	tests := []struct {
		name, host, path string
		vulnerableStatus int
		fixedStatus      int
	}{
		{"current public route decoy", PublicHost, "/v2/invoices/104", 200, 200},
		{"current route keeps Chapter 1 loader", PublicHost, "/v2/invoices/205", 404, 404},
		{"deprecated public route", PublicHost, "/v1/invoices/104", 200, 404},
		{"unlisted staging v1 route", StagingHost, "/v1/invoices/104", 200, 404},
		{"near-identical staging v2 route", StagingHost, "/v2/invoices/104", 404, 404},
		{"unknown preview host decoy", "ledger-preview.internal", "/v1/invoices/104", 404, 404},
	}

	apps := map[Mode]*App{Vulnerable: NewChapter9App(Vulnerable), Fixed: NewChapter9App(Fixed)}
	for _, test := range tests {
		for _, mode := range []Mode{Vulnerable, Fixed} {
			t.Run(string(mode)+"/"+test.name, func(t *testing.T) {
				want := test.vulnerableStatus
				if mode == Fixed {
					want = test.fixedStatus
				}
				got := request(t, apps[mode], test.host, "alice-token", test.path)
				if got.status != want {
					t.Fatalf("status = %d, want %d; body=%q", got.status, want, got.body)
				}
			})
		}
	}
}

func TestChapter09InventoryReconciliation(t *testing.T) {
	wantVulnerable := []string{
		"deprecated-still-serving:GET api.ledger.example/v1/invoices/{id}",
		"deprecated-still-serving:GET api.ledger.example/v1/invoices/{id}/pdf",
		"deprecated-still-serving:GET ledger-staging.internal/v1/invoices/{id}",
		"unlisted-host:ledger-staging.internal",
	}
	if got := ReconcileInventory(Chapter09Inventory(Vulnerable)); !reflect.DeepEqual(got, wantVulnerable) {
		t.Fatalf("vulnerable findings = %#v, want %#v", got, wantVulnerable)
	}
	if got := ReconcileInventory(Chapter09Inventory(Fixed)); len(got) != 0 {
		t.Fatalf("fixed findings = %#v, want none", got)
	}
}

func TestChapter09FixturesMatchRegisteredRoutes(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		fixture := Chapter09Inventory(mode)
		app := NewChapter9App(mode)
		if !reflect.DeepEqual(fixture.Code, app.Routes()) {
			t.Fatalf("%s code fixture = %#v, routes = %#v", mode, fixture.Code, app.Routes())
		}
	}
}
