package ledger

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type chapter07RoundTrip struct {
	requests []string
	reply    func(*http.Request) (int, string, string)
}

func (f *chapter07RoundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, r.URL.String())
	status, location, body := f.reply(r)
	header := make(http.Header)
	if location != "" {
		header.Set("Location", location)
	}
	return &http.Response{StatusCode: status, Header: header,
		Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func chapter07Resolver(answers map[string][]netip.Addr, calls *[]string) resolveIPs {
	return func(_ context.Context, host string) ([]netip.Addr, error) {
		*calls = append(*calls, host)
		if ips, ok := answers[host]; ok {
			return ips, nil
		}
		return nil, errors.New("unlisted test hostname")
	}
}

func chapter07App(mode Mode, answers map[string][]netip.Addr, rt *chapter07RoundTrip, calls *[]string) *App {
	resolver := chapter07Resolver(answers, calls)
	vulnerable := defaultFetch{client: &http.Client{Transport: rt}} // nil CheckRedirect follows 302
	fixed := &egress{resolve: resolver, dial: func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("canned transport should not dial")
	}, transport: rt}
	return buildAppWithNetwork(mode, chapter07, func() time.Time { return chapter06TestTime },
		&chapter07Network{vulnerable: vulnerable, fixed: fixed})
}

func chapter07Request(t *testing.T, app *App, method, path, token, body string) response {
	t.Helper()
	return chapter04Request(t, app, method, path, token, "192.0.2.77", body)
}

func TestChapter07WebhookExerciseCases(t *testing.T) {
	public := netip.MustParseAddr("8.8.8.8")
	report := netip.MustParseAddr("1.1.1.1")
	for _, mode := range []Mode{Vulnerable, Fixed} {
		t.Run(string(mode), func(t *testing.T) {
			// 1: the obvious HTTP literal is a decoy rejected before either fetcher.
			calls := []string{}
			rt := &chapter07RoundTrip{reply: func(*http.Request) (int, string, string) { return 200, "", "token" }}
			app := chapter07App(mode, map[string][]netip.Addr{}, rt, &calls)
			got := chapter07Request(t, app, http.MethodPost, "/v2/webhooks/test", "alice-token", `{"url":"http://169.254.169.254/latest/meta-data/"}`)
			if got.status != 400 || len(rt.requests) != 0 || len(calls) != 0 {
				t.Fatalf("case 1 literal = %+v requests=%v DNS=%v", got, rt.requests, calls)
			}

			// 2: a legitimate public HTTPS destination still works in both modes.
			calls = nil
			rt = &chapter07RoundTrip{reply: func(*http.Request) (int, string, string) { return 200, "", "ok" }}
			app = chapter07App(mode, map[string][]netip.Addr{"hooks.cedar.example": {public}}, rt, &calls)
			got = chapter07Request(t, app, http.MethodPost, "/v2/webhooks/test", "alice-token", `{"url":"https://hooks.cedar.example/ledger"}`)
			if got.status != 200 || !strings.Contains(got.body, `"status":200`) || len(rt.requests) != 1 || rt.requests[0] != "https://hooks.cedar.example/ledger" {
				t.Fatalf("case 2 public = %+v requests=%v", got, rt.requests)
			}
			if mode == Fixed && len(calls) != 1 {
				t.Fatalf("fixed public lookup count = %v", calls)
			}

			// 4 and 5: byte-identical input, opposite public-host replies.
			for _, redirect := range []bool{true, false} {
				calls = nil
				rt = &chapter07RoundTrip{reply: func(r *http.Request) (int, string, string) {
					if r.URL.Hostname() == "169.254.169.254" {
						return 200, "", "metadata fixture"
					}
					if redirect {
						return 302, "http://169.254.169.254/latest/meta-data/", ""
					}
					return 200, "", "ok"
				}}
				app = chapter07App(mode, map[string][]netip.Addr{"report.cedar.example": {report}}, rt, &calls)
				got = chapter07Request(t, app, http.MethodPost, "/v2/webhooks/test", "alice-token", `{"url":"https://report.cedar.example/hook"}`)
				if redirect && mode == Vulnerable && (got.status != 200 || !strings.Contains(got.body, `"status":200`) || len(rt.requests) != 2 || !strings.Contains(rt.requests[1], "169.254.169.254")) {
					t.Fatalf("case 4 vulnerable redirect = %+v requests=%v", got, rt.requests)
				}
				if redirect && mode == Fixed && (got.status != 200 || !strings.Contains(got.body, `"status":302`) || len(rt.requests) != 1) {
					t.Fatalf("case 4 fixed redirect = %+v requests=%v", got, rt.requests)
				}
				if !redirect && (got.status != 200 || !strings.Contains(got.body, `"status":200`) || len(rt.requests) != 1) {
					t.Fatalf("case 5 public reply = %+v requests=%v", got, rt.requests)
				}
				if mode == Fixed && (len(calls) != 1 || calls[0] != "report.cedar.example") {
					t.Fatalf("fixed report resolution = %v", calls)
				}
			}
		})
	}
}

func TestChapter07LinkLocalDialAndPinning(t *testing.T) {
	linkLocal := netip.MustParseAddr("169.254.169.254")
	public := netip.MustParseAddr("8.8.8.8")
	url := `{"url":"https://hooks.cedar.example/ledger"}`
	var attempted []string
	vulnerableTransport := &http.Transport{Proxy: nil, DialContext: func(_ context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || host != "hooks.cedar.example" {
			t.Fatalf("vulnerable dial target = %s, err=%v", address, err)
		}
		attempted = append(attempted, net.JoinHostPort(linkLocal.String(), port))
		return nil, errors.New("synthetic link-local dial stop")
	}}
	queries := []string{}
	fixed := &egress{resolve: chapter07Resolver(map[string][]netip.Addr{"hooks.cedar.example": {linkLocal}}, &queries),
		dial: func(_ context.Context, _, address string) (net.Conn, error) {
			attempted = append(attempted, address)
			return nil, errors.New("must not dial")
		}}
	deps := &chapter07Network{vulnerable: defaultFetch{&http.Client{Transport: vulnerableTransport}}, fixed: fixed}
	vulnerable := buildAppWithNetwork(Vulnerable, chapter07, func() time.Time { return chapter06TestTime }, deps)
	got := chapter07Request(t, vulnerable, http.MethodPost, "/v2/webhooks/test", "alice-token", url)
	if got.status != 502 || len(attempted) != 1 || attempted[0] != "169.254.169.254:443" {
		t.Fatalf("case 3 vulnerable = %+v attempted=%v", got, attempted)
	}
	attempted = nil
	fixedApp := buildAppWithNetwork(Fixed, chapter07, func() time.Time { return chapter06TestTime }, deps)
	got = chapter07Request(t, fixedApp, http.MethodPost, "/v2/webhooks/test", "alice-token", url)
	if got.status != 400 || len(attempted) != 0 || len(queries) != 1 {
		t.Fatalf("case 3 fixed = %+v attempted=%v queries=%v", got, attempted, queries)
	}

	// A canned RoundTripper cannot prove pinning. This direct callback must see the
	// approved address and must not trigger a second hostname lookup.
	queries = nil
	attempted = nil
	pinner := &egress{resolve: chapter07Resolver(map[string][]netip.Addr{"hooks.cedar.example": {public}}, &queries),
		dial: func(_ context.Context, _, address string) (net.Conn, error) {
			attempted = append(attempted, address)
			return nil, errors.New("synthetic public dial stop")
		}}
	if _, err := pinner.Fetch(context.Background(), "https://hooks.cedar.example/ledger"); err == nil || len(queries) != 1 || len(attempted) != 1 || attempted[0] != "8.8.8.8:443" {
		t.Fatalf("pinned dial: err=%v queries=%v attempted=%v", err, queries, attempted)
	}
}

func TestChapter07AddressClasses(t *testing.T) {
	for address, allowed := range map[string]bool{
		"8.8.8.8": true, "1.1.1.1": true, "169.254.169.254": false,
		"127.0.0.1": false, "10.4.5.6": false, "192.168.1.5": false,
		"203.0.113.10": false, "198.51.100.7": false, "192.0.2.1": false,
		"192.88.99.2": false, "100.64.1.1": false, "198.19.1.1": false,
		"::1": false, "fe80::1": false, "fc00::1": false, "2001:db8::1": false,
		"64:ff9b::a9fe:a9fe": false, "2002::1": false, "3fff::1": false,
		"2001:4860::8888%lo": false,
	} {
		if got := publicAddress(netip.MustParseAddr(address)); got != allowed {
			t.Errorf("publicAddress(%s)=%v, want %v", address, got, allowed)
		}
	}
	queries := []string{}
	attempted := false
	e := &egress{resolve: chapter07Resolver(map[string][]netip.Addr{
		"mixed.example": {netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")},
	}, &queries), dial: func(context.Context, string, string) (net.Conn, error) {
		attempted = true
		return nil, errors.New("unexpected dial")
	}}
	if _, err := e.Fetch(context.Background(), "https://mixed.example/"); !errors.Is(err, errEgressDenied) || attempted || len(queries) != 1 {
		t.Fatalf("mixed DNS answer: err=%v attempted=%v queries=%v", err, attempted, queries)
	}
}

func TestChapter07PDFSecondPathAndEarlierRepairs(t *testing.T) {
	public := netip.MustParseAddr("8.8.8.8")
	for _, mode := range []Mode{Vulnerable, Fixed} {
		for _, redirect := range []bool{true, false} {
			queries := []string{}
			rt := &chapter07RoundTrip{reply: func(r *http.Request) (int, string, string) {
				if redirect && r.URL.Hostname() == "logo.cedar.example" {
					return 302, "http://169.254.169.254/logo", ""
				}
				return 200, "", "logo fixture"
			}}
			app := chapter07App(mode, map[string][]netip.Addr{"logo.cedar.example": {public}}, rt, &queries)
			got := chapter07Request(t, app, http.MethodGet, "/v1/invoices/104/pdf", "alice-token", "")
			if got.status != 200 || strings.Contains(got.body, "email") || !strings.Contains(got.body, "LGR-A7") {
				t.Fatalf("%s PDF = %+v", mode, got)
			}
			wantRequests := 1
			if redirect && mode == Vulnerable {
				wantRequests = 2
			}
			if len(rt.requests) != wantRequests || (mode == Fixed && len(queries) != 1) {
				t.Fatalf("%s PDF logo redirect=%v requests=%v DNS=%v", mode, redirect, rt.requests, queries)
			}
			if cross := chapter07Request(t, app, http.MethodGet, "/v1/invoices/205/pdf", "alice-token", ""); cross.status != 404 || len(rt.requests) != wantRequests {
				t.Fatalf("%s cross-tenant PDF = %+v requests=%v", mode, cross, rt.requests)
			}
			if wrongRole := chapter07Request(t, app, http.MethodDelete, "/v2/invoices/104", "alice-token", ""); wrongRole.status != 403 {
				t.Fatalf("%s Chapter 5 role regression = %+v", mode, wrongRole)
			}
			if mobile := chapter07Request(t, app, http.MethodGet, "/v2/me", mobileAppKey, ""); mobile.status != 401 {
				t.Fatalf("%s Chapter 2 identity regression = %+v", mode, mobile)
			}
			if noAuth := chapter07Request(t, app, http.MethodPost, "/v2/webhooks/test", "", `{"url":"https://logo.cedar.example/mark.png"}`); noAuth.status != 401 {
				t.Fatalf("%s webhook anonymous = %+v", mode, noAuth)
			}
		}
	}
}

func TestChapter07PDFLinkLocalAnswer(t *testing.T) {
	linkLocal := netip.MustParseAddr("169.254.169.254")
	for _, mode := range []Mode{Vulnerable, Fixed} {
		attempted := []string{}
		queries := []string{}
		vulnerableTransport := &http.Transport{Proxy: nil, DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(address)
			if err != nil {
				t.Fatalf("bad logo dial address: %v", err)
			}
			attempted = append(attempted, net.JoinHostPort(linkLocal.String(), port))
			return nil, errors.New("synthetic logo dial stop")
		}}
		fixed := &egress{resolve: chapter07Resolver(map[string][]netip.Addr{"logo.cedar.example": {linkLocal}}, &queries),
			dial: func(_ context.Context, _, address string) (net.Conn, error) {
				attempted = append(attempted, address)
				return nil, errors.New("must not dial")
			}}
		app := buildAppWithNetwork(mode, chapter07, func() time.Time { return chapter06TestTime },
			&chapter07Network{vulnerable: defaultFetch{&http.Client{Transport: vulnerableTransport}}, fixed: fixed})
		got := chapter07Request(t, app, http.MethodGet, "/v1/invoices/104/pdf", "alice-token", "")
		if got.status != 200 {
			t.Fatalf("%s PDF with denied logo = %+v", mode, got)
		}
		if mode == Vulnerable && (len(attempted) != 1 || attempted[0] != "169.254.169.254:443") {
			t.Fatalf("vulnerable PDF link-local attempt = %v", attempted)
		}
		if mode == Fixed && (len(attempted) != 0 || len(queries) != 1) {
			t.Fatalf("fixed PDF should refuse before dial: attempted=%v queries=%v", attempted, queries)
		}
	}
}
