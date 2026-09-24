package ledger

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type scopedInvoiceKey struct{}
type scopedUserKey struct{}

func registerChapter05Routes(mux *http.ServeMux, routes []Route, store *Store, sessions *sessionStore, mode Mode) []Route {
	added := []Route{
		{Method: http.MethodDelete, Pattern: "/v2/invoices/{id}"},
		{Method: http.MethodPost, Pattern: "/v2/admin/invoices/{id}/void"},
		{Method: http.MethodGet, Pattern: "/v2/admin/users"},
		{Method: http.MethodPatch, Pattern: "/v2/admin/users/{id}"},
	}
	deleteHandler := http.HandlerFunc(fixedChapter05Delete(store))
	if mode == Vulnerable {
		deleteHandler = vulnerableChapter05Delete(store)
	}
	mux.HandleFunc("DELETE /v2/invoices/{id}", deleteHandler)
	mux.HandleFunc("POST /v2/admin/invoices/{id}/void", chapter05Void(store))
	mux.HandleFunc("GET /v2/admin/users", chapter05Users(sessions))
	mux.HandleFunc("PATCH /v2/admin/users/{id}", chapter05UserPatch(sessions))
	return declareChapter05Routes(append(routes, added...))
}

func chapter05Authorization(mode Mode, mux *http.ServeMux, routes []Route, store *Store, sessions *sessionStore, next http.Handler) http.Handler {
	accessByPattern := make(map[string]Access, len(routes))
	for _, route := range routes {
		accessByPattern[route.Method+" "+route.Pattern] = route.Access
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		access, registered := accessByPattern[pattern]
		if !registered {
			next.ServeHTTP(w, r) // ServeMux supplies its normal 404 or 405.
			return
		}
		var guarded http.Handler = requireAccess(access, mode, next)
		if strings.Contains(pattern, "/invoices/{id}") {
			guarded = scopeInvoice(store, guarded)
		} else if pattern == "PATCH /v2/admin/users/{id}" {
			guarded = scopeAdminUser(sessions, guarded)
		} else if pattern == "POST /v2/refunds/{quote}/approve" {
			guarded = scopeQuote(store, guarded)
		}
		guarded.ServeHTTP(w, r)
	})
}

func scopeInvoice(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		var id int
		foundID := false
		for i, part := range parts {
			if part == "invoices" && i+1 < len(parts) {
				id, _ = strconv.Atoi(parts[i+1])
				foundID = true
				break
			}
		}
		invoice, found := store.LoadInvoiceFor(user, id)
		if !foundID || !found || id <= 0 {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopedInvoiceKey{}, invoice)))
	})
}

func scopeAdminUser(sessions *sessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		target, found := sessions.userByName(parts[len(parts)-1])
		if !found || target.Tenant != caller.Tenant {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopedUserKey{}, target)))
	})
}

// excerpt: ch05-require-access
func requireAccess(level Access, mode Mode, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if level == Public {
			next.ServeHTTP(w, r)
			return
		}
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if level == TenantAdmin && mode == Fixed && user.Role != "tenant-admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// end excerpt

// excerpt: ch05-vulnerable-delete
func vulnerableChapter05Delete(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := currentUser(r)
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if _, found := store.LoadInvoiceFor(user, id); !found {
			http.NotFound(w, r)
			return
		}
		delete(store.invoices, id) // tenant scope alone grants a destructive action
		w.WriteHeader(http.StatusNoContent)
	}
}

// end excerpt

func fixedChapter05Delete(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inv := r.Context().Value(scopedInvoiceKey{}).(Invoice)
		delete(store.invoices, inv.ID)
		w.WriteHeader(http.StatusNoContent)
	}
}

func chapter05Void(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inv := r.Context().Value(scopedInvoiceKey{}).(Invoice)
		inv.Status = "void"
		store.invoices[inv.ID] = inv
		w.WriteHeader(http.StatusNoContent)
	}
}

func chapter05Users(sessions *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, _ := currentUser(r)
		users := sessions.usersFor(caller.Tenant)
		sort.Slice(users, func(i, j int) bool { return users[i].Name < users[j].Name })
		writeJSON(w, users)
	}
}

func chapter05UserPatch(sessions *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.Context().Value(scopedUserKey{}).(User)
		var patch struct {
			Role string `json:"role"`
		}
		if decodeOneJSON(r, &patch) != nil || (patch.Role != "user" && patch.Role != "tenant-admin") {
			http.Error(w, "invalid role", http.StatusBadRequest)
			return
		}
		target.Role = patch.Role
		sessions.updateUser(target.Name, target)
		writeJSON(w, target)
	}
}
