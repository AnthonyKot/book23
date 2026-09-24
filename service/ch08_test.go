package ledger

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func chapter08App(mode Mode, override *Settings) *App {
	return buildAppWithSettings(mode, chapter08, func() time.Time { return chapter06TestTime }, nil, override)
}

func TestChapter08ExerciseCases(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			app := chapter08App(mode, nil)
			request := func(path, token string) response {
				return chapter05Request(t, app, http.MethodGet, path, token, "")
			}
			// A: an intentionally public liveness probe is harmless.
			if got := request("/v2/health", ""); got.status != 200 || got.body != "{\"ok\":true}\n" {
				t.Fatalf("A health = %+v", got)
			}
			// B and B': same admin path, opposite declarations and response bodies.
			anonymous := request("/v2/admin/health", "")
			dana := request("/v2/admin/health", "dana-token")
			if mode == Vulnerable {
				if anonymous.status != 200 || dana.status != 200 || !strings.Contains(anonymous.body, StagingHost) || !strings.Contains(anonymous.body, `"debug":true`) || anonymous.body != dana.body {
					t.Fatalf("B/B' vulnerable: anon=%+v Dana=%+v", anonymous, dana)
				}
			} else if anonymous.status != 401 || dana.status != 200 || !strings.Contains(dana.body, `"version"`) || strings.Contains(dana.body, `"hosts"`) || strings.Contains(dana.body, `"debug"`) {
				t.Fatalf("B/B' fixed: anon=%+v Dana=%+v", anonymous, dana)
			}
			// C: the invoice loader still denies 205, but Debug leaks its verdict.
			cross := request("/v2/invoices/205", "alice-token")
			if mode == Vulnerable {
				if cross.status != 404 || !strings.Contains(cross.body, `"reason":"invoice 205 belongs to tenant birch"`) {
					t.Fatalf("C vulnerable = %+v", cross)
				}
			} else if cross.status != 404 || cross.body != "{\"error\":\"not found\"}\n" {
				t.Fatalf("C fixed = %+v", cross)
			}
			// C': a missing number carries no existence or tenant detail in the fixed build.
			missing := request("/v2/invoices/999", "alice-token")
			if missing.status != 404 || mode == Fixed && missing.body != cross.body {
				t.Fatalf("C' missing = %+v", missing)
			}
		})
	}
	// C' with Debug false is separately instantiated in the vulnerable route build.
	settings := StagingSettings()
	settings.Debug = false
	app := chapter08App(Vulnerable, &settings)
	if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/205", "alice-token", ""); got.status != 404 || got.body != "{\"error\":\"not found\"}\n" {
		t.Fatalf("C' Debug false = %+v", got)
	}
}

// excerpt: ch08-settings-test
func TestChapter08ProductionSettings(t *testing.T) {
	app := NewChapter8App(Fixed)
	settings := ProductionSettings()
	if err := validateProductionSettings(settings, app.Routes()); err != nil {
		t.Fatal(err)
	}
	if settings.Debug || len(settings.CORSOrigins) != 1 || settings.CORSOrigins[0] == "*" {
		t.Fatalf("unsafe production settings: %+v", settings)
	}
	got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/205", "alice-token", "")
	if got.status != 404 || got.body != "{\"error\":\"not found\"}\n" {
		t.Fatalf("production denial = %+v", got)
	}
	bad := settings
	bad.Debug = true
	if err := validateProductionSettings(bad, app.Routes()); err == nil {
		t.Fatal("production Debug:true was accepted")
	}
	bad = settings
	bad.CORSOrigins = []string{"*"}
	if err := validateProductionSettings(bad, app.Routes()); err == nil {
		t.Fatal("wildcard CORS origin was accepted")
	}
	vulnerable := NewChapter8App(Vulnerable)
	bad = settings
	if err := validateProductionSettings(bad, vulnerable.Routes()); err == nil || !strings.Contains(err.Error(), "/v2/admin/health") {
		t.Fatalf("unlisted public admin route = %v", err)
	}
}

// end excerpt

func TestChapter08EarlierRepairs(t *testing.T) {
	for _, mode := range []Mode{Vulnerable, Fixed} {
		app := NewChapter8App(mode)
		if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/104", "alice-token", ""); got.status != 200 || strings.Contains(got.body, "collections_note") {
			t.Fatalf("view repair %s = %+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", ""); got.status != 403 {
			t.Fatalf("role repair %s = %+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v2/me", mobileAppKey, ""); got.status != 401 {
			t.Fatalf("identity repair %s = %+v", mode, got)
		}
		if got := chapter05Request(t, app, http.MethodGet, "/v2/invoices/205", "alice-token", ""); got.status != 404 {
			t.Fatalf("loader repair %s = %+v", mode, got)
		}
	}
}
