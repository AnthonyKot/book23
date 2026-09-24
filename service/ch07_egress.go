package ledger

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

var errEgressDenied = errors.New("outbound destination denied")

type resolveIPs func(context.Context, string) ([]netip.Addr, error)
type dialAddress func(context.Context, string, string) (net.Conn, error)

type outboundFetcher interface {
	Fetch(context.Context, string) (*http.Response, error)
}

type egress struct {
	resolve   resolveIPs
	dial      dialAddress
	transport http.RoundTripper // test-only canned responses; nil uses a pinned socket
}

func newEgress() *egress {
	return &egress{
		resolve: func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		},
		dial: (&net.Dialer{}).DialContext,
	}
}

// Conservative snapshot of special-purpose ranges from IANA's IPv4 and IPv6
// registries. Refresh before using this teaching policy in a deployed service.
// https://www.iana.org/assignments/iana-ipv4-special-registry
// https://www.iana.org/assignments/iana-ipv6-special-registry
var deniedAddressRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"), netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("192.175.48.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"), netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"), netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fc00::/7"), netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("ff00::/8"),
}

func publicAddress(ip netip.Addr) bool {
	if !ip.IsValid() || ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, block := range deniedAddressRanges {
		if block.Contains(ip) {
			return false
		}
	}
	return true
}

// excerpt: ch07-egress-policy
func (e *egress) Fetch(ctx context.Context, rawURL string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return nil, errEgressDenied
	}
	host := strings.ToLower(u.Hostname())
	addresses := []netip.Addr{}
	if literal, err := netip.ParseAddr(host); err == nil {
		addresses = append(addresses, literal)
	} else {
		addresses, err = e.resolve(ctx, host) // exactly one DNS lookup
		if err != nil {
			return nil, err
		}
	}
	if len(addresses) == 0 {
		return nil, errEgressDenied
	}
	for _, ip := range addresses {
		if !publicAddress(ip) { // reject the entire answer set
			return nil, errEgressDenied
		}
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	pinned := net.JoinHostPort(addresses[0].Unmap().String(), port)
	transport := http.RoundTripper(&http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return e.dial(ctx, network, pinned) // URL host remains for TLS name validation
	}})
	if e.transport != nil {
		transport = e.transport // canned test replies; dial pinning is tested separately
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errEgressDenied
	}
	return client.Do(req)
}

// end excerpt

type defaultFetch struct{ client *http.Client }

func (f defaultFetch) Fetch(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return f.client.Do(req) // default client follows redirects and resolves during dial
}

func stage07Fetcher(mode Mode, injected *chapter07Network) outboundFetcher {
	if injected == nil {
		injected = &chapter07Network{vulnerable: defaultFetch{http.DefaultClient}, fixed: newEgress()}
	}
	if injected.vulnerable == nil || injected.fixed == nil {
		panic("ledger: Chapter 7 requires both outbound clients")
	}
	if mode == Vulnerable {
		return injected.vulnerable
	}
	return injected.fixed
}

type chapter07Network struct {
	vulnerable outboundFetcher
	fixed      outboundFetcher
}
