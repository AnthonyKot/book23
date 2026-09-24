package ledger

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type RouteKey struct{ Method, Pattern string }

type Settings struct {
	Debug        bool
	CORSOrigins  []string
	PublicRoutes []RouteKey
	Hosts        []string
}

// excerpt: ch08-production-settings
func ProductionSettings() Settings {
	return Settings{
		Debug:       false,
		CORSOrigins: []string{"https://app.ledger.example"},
		PublicRoutes: []RouteKey{
			{http.MethodGet, "/v2/health"},
			{http.MethodPost, "/v2/auth/login"},
			{http.MethodPost, "/v2/auth/otp/request"},
			{http.MethodPost, "/v2/auth/otp/verify"},
		},
		Hosts: []string{PublicHost},
	}
}

func StagingSettings() Settings {
	return Settings{
		Debug:       true,
		CORSOrigins: []string{"https://app.ledger.example"},
		PublicRoutes: []RouteKey{
			{http.MethodGet, "/v2/health"},
			{http.MethodPost, "/v2/auth/login"},
			{http.MethodPost, "/v2/auth/otp/request"},
			{http.MethodPost, "/v2/auth/otp/verify"},
		},
		Hosts: []string{PublicHost, StagingHost},
	}
}

// end excerpt

func validateProductionSettings(s Settings, routes []Route) error {
	if s.Debug {
		return fmt.Errorf("production debug enabled")
	}
	for _, origin := range s.CORSOrigins {
		if origin == "*" {
			return fmt.Errorf("wildcard CORS origin")
		}
	}
	allowed := make(map[RouteKey]bool, len(s.PublicRoutes))
	for _, key := range s.PublicRoutes {
		allowed[key] = true
	}
	seen := make(map[RouteKey]bool, len(s.PublicRoutes))
	for _, route := range routes {
		key := RouteKey{route.Method, route.Pattern}
		if route.Access == Public && !allowed[key] {
			return fmt.Errorf("unlisted public route: %s %s", route.Method, route.Pattern)
		}
		if route.Access == Public {
			seen[key] = true
		}
	}
	for key := range allowed {
		if !seen[key] {
			return fmt.Errorf("allow-listed route missing or not public: %s %s", key.Method, key.Pattern)
		}
	}
	return nil
}

// excerpt: ch08-vulnerable-health-decl
var vulnerableHealthDeclaration = Route{Method: http.MethodGet, Pattern: "/v2/admin/health", Access: Public}

// end excerpt

// excerpt: ch08-fixed-health-decl
var fixedHealthDeclaration = Route{Method: http.MethodGet, Pattern: "/v2/admin/health", Access: TenantAdmin}

// end excerpt

func registerChapter08Routes(mux *http.ServeMux, routes []Route, mode Mode, settings Settings) []Route {
	health := Route{Method: http.MethodGet, Pattern: "/v2/health", Access: Public}
	admin := fixedHealthDeclaration
	if mode == Vulnerable {
		admin = vulnerableHealthDeclaration
	}
	mux.HandleFunc("GET /v2/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, struct {
			OK bool `json:"ok"`
		}{true})
	})
	mux.HandleFunc("GET /v2/admin/health", func(w http.ResponseWriter, _ *http.Request) {
		if mode == Vulnerable {
			writeJSON(w, struct {
				Version string   `json:"version"`
				Commit  string   `json:"commit"`
				Hosts   []string `json:"hosts"`
				Debug   bool     `json:"debug"`
			}{"fixture", "fixture", settings.Hosts, settings.Debug})
			return
		}
		writeJSON(w, struct {
			Version string `json:"version"`
		}{"fixture"})
	})
	policy := append(append(append([]Route{}, chapter05Declarations...), chapter06Declarations...), chapter07Declarations...)
	policy = append(policy, health, admin)
	return declareRoutes(append(routes, health, Route{Method: admin.Method, Pattern: admin.Pattern}), policy)
}

type chapter08Capture struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (c *chapter08Capture) Header() http.Header { return c.header }
func (c *chapter08Capture) WriteHeader(status int) {
	if c.status == 0 {
		c.status = status
	}
}
func (c *chapter08Capture) Write(p []byte) (int, error) {
	if c.status == 0 {
		c.status = 200
	}
	return c.body.Write(p)
}

func (s *Store) invoiceMissReason(user User, id int) string {
	row, found := s.invoices[id]
	if found && row.Tenant != user.Tenant {
		return fmt.Sprintf("invoice %d belongs to tenant %s", id, strings.ToLower(row.Tenant))
	}
	return "invoice not found"
}

// excerpt: ch08-debug-error-writer
func chapter08ErrorWriter(settings Settings, store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/v2/invoices/") {
			next.ServeHTTP(w, r)
			return
		}
		capture := &chapter08Capture{header: make(http.Header)}
		next.ServeHTTP(capture, r)
		if capture.status == http.StatusNotFound {
			body := map[string]string{"error": "not found"}
			if settings.Debug {
				user, ok := currentUser(r)
				id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/v2/invoices/"))
				if ok && err == nil {
					body["reason"] = store.invoiceMissReason(user, id)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, body)
			return
		}
		for key, values := range capture.header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		if capture.status != 0 {
			w.WriteHeader(capture.status)
		}
		_, _ = w.Write(capture.body.Bytes())
	})
}

// end excerpt
