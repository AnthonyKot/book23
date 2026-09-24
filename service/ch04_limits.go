package ledger

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	otpIPBudget     = 8 // fixture-only noisy-client guard, distinct from the target budget
	otpTargetBudget = 5
	otpWindow       = 10 * time.Minute
	lookupBudget    = 30
	lookupWindow    = time.Minute
	maximumPageSize = 50
	fixtureOTPCode  = "731842" // simulates a texted code; never returned by the request route
)

type limitBucket struct {
	start time.Time
	count int
}

type windowLimiter struct {
	mu      sync.Mutex
	clock   func() time.Time
	window  time.Duration
	buckets map[string]limitBucket
}

func newWindowLimiter(clock func() time.Time, window time.Duration) *windowLimiter {
	return &windowLimiter{clock: clock, window: window, buckets: make(map[string]limitBucket)}
}

func (l *windowLimiter) allow(key string, budget int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	bucket := l.buckets[key]
	if bucket.start.IsZero() || now.Before(bucket.start) || !now.Before(bucket.start.Add(l.window)) {
		bucket = limitBucket{start: now}
	}
	if bucket.count >= budget {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}

func keyPerIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// excerpt: ch04-limiter-per-ip
func perIPGuard(limiter *windowLimiter, budget int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(keyPerIP(r), budget) {
			http.Error(w, "IP rate limit", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// end excerpt

func keyPerChallenge(account, challengeID string) string {
	return strings.ToLower(account) + ":" + challengeID
}

func keyPerTenantRouteMinute(tenant, route string) string {
	return strings.ToLower(tenant) + ":" + route
}

type otpChallenge struct {
	AccountID   string
	ChallengeID string
	CodeHash    [32]byte
	ExpiresAt   time.Time
	Attempts    int
	Locked      bool
	Used        bool
}

type challengeStore struct {
	mu         sync.Mutex
	clock      func() time.Time
	next       int
	active     map[string]string
	challenges map[string]*otpChallenge
}

func newChallengeStore(clock func() time.Time) *challengeStore {
	return &challengeStore{clock: clock, active: make(map[string]string), challenges: make(map[string]*otpChallenge)}
}

func (s *challengeStore) request(account string, mode Mode) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account = strings.ToLower(account)
	if id := s.active[account]; id != "" && mode == Fixed {
		challenge := s.challenges[keyPerChallenge(account, id)]
		if challenge != nil && s.clock().Before(challenge.ExpiresAt) {
			if challenge.Locked {
				return "", false
			}
			if !challenge.Used {
				return id, true // resend without resetting the target budget
			}
		}
	}
	s.next++
	id := fmt.Sprintf("otp-%d", s.next)
	s.challenges[keyPerChallenge(account, id)] = &otpChallenge{
		AccountID: account, ChallengeID: id, CodeHash: sha256.Sum256([]byte(fixtureOTPCode)),
		ExpiresAt: s.clock().Add(otpWindow),
	}
	s.active[account] = id
	return id, true
}

type otpVerifyRequest struct {
	AccountID   string `json:"account_id"`
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type otpOutcome int

const (
	otpWrong otpOutcome = iota
	otpMatched
	otpLocked
	otpExpired
)

func (s *challengeStore) current(input otpVerifyRequest) *otpChallenge {
	account := strings.ToLower(input.AccountID)
	if s.active[account] != input.ChallengeID {
		return nil
	}
	return s.challenges[keyPerChallenge(account, input.ChallengeID)]
}

// excerpt: ch04-otp-vulnerable
func (s *challengeStore) verifyVulnerable(input otpVerifyRequest) otpOutcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge := s.current(input)
	if challenge == nil || challenge.Used || !s.clock().Before(challenge.ExpiresAt) {
		return otpExpired
	}
	if sha256.Sum256([]byte(input.Code)) != challenge.CodeHash {
		return otpWrong // no counter against this account's challenge
	}
	challenge.Used = true
	return otpMatched
}

// end excerpt

// excerpt: ch04-otp-fixed
func (s *challengeStore) verifyFixed(input otpVerifyRequest) otpOutcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge := s.current(input)
	if challenge == nil || challenge.Used || !s.clock().Before(challenge.ExpiresAt) {
		return otpExpired
	}
	if challenge.Locked {
		return otpLocked
	}
	if sha256.Sum256([]byte(input.Code)) != challenge.CodeHash {
		challenge.Attempts++
		if challenge.Attempts >= otpTargetBudget {
			challenge.Locked = true
			return otpLocked
		}
		return otpWrong
	}
	challenge.Used = true
	return otpMatched
}

// end excerpt

type chapter04Limits struct {
	clock         func() time.Time
	challenges    *challengeStore
	otpIP         *windowLimiter
	lookupIP      *windowLimiter
	lookupGateway *windowLimiter
	lookupHandler *windowLimiter
}

func newChapter04Limits(clock func() time.Time) *chapter04Limits {
	return &chapter04Limits{
		clock: clock, challenges: newChallengeStore(clock),
		otpIP: newWindowLimiter(clock, lookupWindow), lookupIP: newWindowLimiter(clock, lookupWindow),
		lookupGateway: newWindowLimiter(clock, lookupWindow), lookupHandler: newWindowLimiter(clock, lookupWindow),
	}
}

func otpRequestHandler(mode Mode, challenges *challengeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			AccountID string `json:"account_id"`
		}
		if decodeOneJSON(r, &input) != nil {
			http.Error(w, "invalid OTP request", http.StatusBadRequest)
			return
		}
		if _, ok := namedUser(input.AccountID); !ok {
			http.Error(w, "unknown account", http.StatusBadRequest)
			return
		}
		id, allowed := challenges.request(input.AccountID, mode)
		if !allowed {
			http.Error(w, "locked", http.StatusLocked)
			return
		}
		writeJSON(w, struct {
			ChallengeID string `json:"challenge_id"`
		}{id})
	}
}

func otpVerifyHandler(mode Mode, challenges *challengeStore, sessions *sessionStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input otpVerifyRequest
		if decodeOneJSON(r, &input) != nil || len(input.Code) != 6 {
			http.Error(w, "invalid OTP verify request", http.StatusBadRequest)
			return
		}
		user, found := namedUser(input.AccountID)
		if !found {
			http.Error(w, "invalid OTP verify request", http.StatusBadRequest)
			return
		}
		outcome := otpExpired
		if mode == Vulnerable {
			outcome = challenges.verifyVulnerable(input)
		} else {
			outcome = challenges.verifyFixed(input)
		}
		switch outcome {
		case otpMatched:
			writeJSON(w, struct {
				Token string `json:"token"`
			}{sessions.issue(user)})
		case otpLocked:
			http.Error(w, "locked", http.StatusLocked)
		default:
			http.Error(w, "wrong or expired code", http.StatusBadRequest)
		}
	})
}

var gatewayLookupRoutes = map[Mode]map[string]bool{
	Vulnerable: {"/v2/invoices": true},
	Fixed:      {"/v2/invoices": true, "/v1/invoices": true},
}

func chapter04Gateway(mode Mode, limits *chapter04Limits, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Has("email") && gatewayLookupRoutes[mode][r.URL.Path] {
			if mode == Vulnerable {
				if !limits.lookupIP.allow(keyPerIP(r), lookupBudget) {
					http.Error(w, "IP lookup limit", http.StatusTooManyRequests)
					return
				}
			} else if user, ok := currentUser(r); ok {
				key := keyPerTenantRouteMinute(user.Tenant, r.URL.Path)
				if !limits.lookupGateway.allow(key, lookupBudget) {
					http.Error(w, "gateway lookup limit", http.StatusTooManyRequests)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func chapter04Lookup(store *Store, mode Mode, limits *chapter04Limits, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Error(w, "email is required", http.StatusBadRequest)
			return
		}
		if mode == Fixed {
			key := keyPerTenantRouteMinute(user.Tenant, "/"+version+"/invoices")
			if !limits.lookupHandler.allow(key, lookupBudget) {
				http.Error(w, "handler lookup limit", http.StatusTooManyRequests)
				return
			}
		}
		views := make([]any, 0)
		for _, invoice := range store.InvoicesFor(user) {
			if strings.EqualFold(invoice.Customer.Email, email) {
				views = append(views, ViewFor(user, invoice))
			}
		}
		writeJSON(w, views)
	}
}

func chapter04Invoices(store *Store, mode Mode, limits *chapter04Limits) http.HandlerFunc {
	lookup := chapter04Lookup(store, mode, limits, "v2")
	tenantList := chapter03List(store, Fixed, true)
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("email") {
			lookup(w, r)
			return
		}
		if query.Has("limit") {
			user, ok := currentUser(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			n, err := strconv.Atoi(query.Get("limit"))
			if err != nil || n <= 0 {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			if mode == Fixed && n > maximumPageSize {
				n = maximumPageSize
			}
			invoices := store.InvoicesFor(user)
			if n > len(invoices) {
				n = len(invoices)
			}
			views := make([]any, n)
			for i := 0; i < n; i++ {
				views[i] = ViewFor(user, invoices[i])
			}
			writeJSON(w, views)
			return
		}
		tenantList(w, r)
	}
}
