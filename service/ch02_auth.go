package ledger

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	mobileAppKey = "mk_ledger_mobile_demo"
	cedarKey     = "ck_cedar_demo"
	birchKey     = "bk_birch_demo"
	sessionLife  = 12 * time.Hour
)

type session struct {
	User     User
	IssuedAt time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	clock    func() time.Time
	sessions map[string]session
	next     int
}

func newSessionStore(clock func() time.Time) *sessionStore {
	if clock == nil {
		panic("ledger: nil clock")
	}
	return &sessionStore{clock: clock, sessions: map[string]session{
		"alice-token": {User: usersByToken["alice-token"], IssuedAt: clock()},
		"ben-token":   {User: usersByToken["ben-token"], IssuedAt: clock()},
		"dana-token":  {User: usersByToken["dana-token"], IssuedAt: clock()},
	}}
}

func (s *sessionStore) issue(user User) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	token := "fixture-session-" + strconv.Itoa(s.next)
	s.sessions[token] = session{User: user, IssuedAt: s.clock()}
	return token
}

func (s *sessionStore) lookup(token string, enforceExpiry bool) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.sessions[token]
	now := s.clock()
	if !ok || (enforceExpiry && (now.Before(record.IssuedAt) || !now.Before(record.IssuedAt.Add(sessionLife)))) {
		return User{}, false
	}
	return record.User, true
}

func (s *sessionStore) revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *sessionStore) updateUser(originalName string, user User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, record := range s.sessions {
		if record.User.Name == originalName {
			record.User = user
			s.sessions[token] = record
		}
	}
}

type requestPrincipal struct {
	user          User
	authenticated bool
}

type principalContextKey struct{}

func newChapter2AppAt(mode Mode, clock func() time.Time) *App {
	return buildAppWithClock(mode, chapter02, clock)
}

func chapter02Identity(mode Mode, sessions *sessionStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var user User
		var ok bool
		if mode == Vulnerable {
			user, ok = vulnerableCurrentUser(r, sessions)
		} else {
			user, ok = fixedCurrentUser(r, sessions)
		}
		ctx := context.WithValue(r.Context(), principalContextKey{}, requestPrincipal{user, ok})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}

func namedUser(name string) (User, bool) {
	for _, user := range usersByToken {
		if strings.EqualFold(user.Name, name) {
			return user, true
		}
	}
	return User{}, false
}

func tenantServiceAccount(token string) (User, bool) {
	switch token {
	case cedarKey:
		return User{Name: "Cedar service account", Tenant: "Cedar", Role: "service"}, true
	case birchKey:
		return User{Name: "Birch service account", Tenant: "Birch", Role: "service"}, true
	default:
		return User{}, false
	}
}

// excerpt: ch02-vulnerable
func vulnerableCurrentUser(r *http.Request, sessions *sessionStore) (User, bool) {
	token := bearerToken(r)
	if user, ok := sessions.lookup(token, false); ok {
		if claimed, found := namedUser(r.Header.Get("X-User")); found {
			return claimed, true // support override trusts a caller-written header
		}
		return user, true
	}
	if token == mobileAppKey {
		return namedUser(r.Header.Get("X-User"))
	}
	if account, ok := tenantServiceAccount(token); ok {
		if claimed, found := namedUser(r.Header.Get("X-User")); found && claimed.Tenant == account.Tenant {
			return claimed, true // tenant key can impersonate a person
		}
		return account, true
	}
	return User{}, false
}

// end excerpt

// excerpt: ch02-fixed
func fixedCurrentUser(r *http.Request, sessions *sessionStore) (User, bool) {
	token := bearerToken(r)
	if user, ok := sessions.lookup(token, true); ok {
		return user, true // identity comes only from a live server-side session
	}
	if account, ok := tenantServiceAccount(token); ok {
		return account, true // the key names a tenant integration, not a person
	}
	return User{}, false // the shared mobile app key is not a login
}

// end excerpt

func loginHandler(sessions *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			User     string `json:"user"`
			Password string `json:"password"`
		}
		if decodeOneJSON(r, &input) != nil {
			http.Error(w, "invalid login", http.StatusBadRequest)
			return
		}
		user, ok := namedUser(input.User)
		if !ok || input.Password != "fixture-"+strings.ToLower(user.Name) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		writeJSON(w, struct {
			Token string `json:"token"`
		}{sessions.issue(user)})
	}
}

func identityProbe(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, user)
}
