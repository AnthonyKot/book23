<!-- Incident claims: checks/claims/07.tsv (archived HackerOne report and Shopify engineering recap). Ledger's webhook and logo fetch are invented analogies. -->

# The Server That Fetched for You

In April 2018 a researcher, `0xacb`, reported a bug to Shopify that did not touch a single one of
Shopify's own servers by name. Shopify Exchange, the marketplace where people bought and sold
storefronts, showed a screenshot of each store on its listing page. To make the picture, a Shopify
server loaded the seller's store in a headless browser and photographed it. The seller controlled
that store, including its template. So the researcher put one line in the template that told the
browser to navigate somewhere else: `window.location` pointed at the cloud provider's metadata
service, the address every instance can reach to read its own credentials.

The screenshot browser followed that script-driven navigation and photographed what it found
there. On the listing page appeared a PNG of the instance's service-account token. The metadata service normally guards
itself with a required header, but an older `/v1beta1` path returned the same token without it.
The researcher then made separate screenshot requests for the instance's `kube-env` data, including
a certificate and key that helped reach the cluster and, eventually, a root shell inside
containers in that slice of Shopify's infrastructure.

The request went out from Shopify's network, so it carried Shopify's trust. Nothing the researcher
could send from the outside would have reached the metadata service; the screenshot server reached
it on the researcher's behalf. Shopify disabled the service within an hour, then deployed a metadata
concealment proxy and cut off access to internal IPs across the affected infrastructure. They paid
$25,000, rated as a core remote-code-execution finding, though the vulnerable slice did not include
Shopify core.

The class is **server-side request forgery**, OWASP API7: the server can be made to issue an
outbound request to a destination the caller influences, and that request reaches somewhere the
caller could not reach directly. Exchange's surface — a screenshot renderer driven by a page the
seller controlled — is not Ledger's. Ledger has no browser to hijack, and its redirect below is an
HTTP 302 rather than `window.location`. But the mechanism is the same
the moment any Ledger route fetches a URL that a caller chose.

## Where the bug lives: a URL from the request, a fetch with no boundary

Ledger lets a tenant register a webhook so it hears about paid invoices. Cedar's is
`https://hooks.cedar.example/ledger`. Because operators mistype these, there is a convenience route:
`POST /v2/webhooks/test`, which fetches the URL in the body and reports the status it got back. Here
is that handler before this chapter's fix.

```go
func vulnerableChapter07Webhook(fetcher outboundFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawURL, ok := chapter07WebhookInput(w, r) // checks only the submitted URL
		if !ok {
			return
		}
		reply, err := fetcher.Fetch(r.Context(), rawURL) // default client resolves and redirects
		writeWebhookResult(w, reply, err)
	}
}
```

Read it the way `0xacb` read Exchange. The handler does check the URL: it must parse, it must be
`https`, and it rejects the obvious literal `http://169.254.169.254/`. Then it hands the string to
an HTTP client with `http.Get`'s default behavior, which does three things the check never saw.
It resolves the hostname to an address — and DNS can answer with a private one. It opens a
connection to whatever that address is.
And if the response is a redirect, it follows it, to a new URL that passed no check at all. The
value the handler inspected and the value the socket connected to are not the same value.

So a caller submits `https://hooks.cedar.example/ledger`, which looks exactly like Cedar's real
webhook, and it passes. If the name resolves to `169.254.169.254`, the client opens a connection
to the metadata service on the caller's say-so; the metadata service does not speak TLS, so the
handshake fails, but the boundary has already been crossed, and an internal service that does
speak TLS would answer. If instead a public server answers
`302 Location: http://169.254.169.254/latest/meta-data/`, the default client follows it, plain
HTTP now, and reaches the metadata service. The test route reports only the status it got back,
so what leaks here is the fact of the fetch, not a token; the delivery path, which forwards the
body, is the same client. The check guarded the string. The attack lives in the resolve, the
connect, and the redirect.

The fix moves the decision to those three moments and refuses to leave them:

```go
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
```

The webhook test now goes through it, and nothing about the handler's own logic changes except the
call it makes:

```go
func fixedChapter07Webhook(fetcher outboundFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawURL, ok := chapter07WebhookInput(w, r)
		if !ok {
			return
		}
		reply, err := fetcher.Fetch(r.Context(), rawURL) // shared egress checks the socket target
		writeWebhookResult(w, reply, err)
	}
}
```

Three details decide whether this is real.

- **The address is resolved, then checked, then dialed — in that order, with the checked address
  pinned.** Classifying the hostname and then letting the HTTP client resolve it again invites a
  second, different answer between the two lookups. `egress` resolves once, rejects any private,
  loopback, or link-local address, and dials the address it approved.
- **Redirects are not followed.** A 200 from a public host and a 302 from that same host are
  different events; the second one is a new fetch to a place the policy never saw. The client
  returns the redirect as a result to display, and stops.
- **The allow-list is by resolved address class, not by hostname.** Trusting `hooks.cedar.example`
  by name is worthless: its DNS can be pointed inward, and a redirect leaves the host entirely. The
  only durable statement is "the socket connected to a public address."

## If it is one wrapper, why is it hard?

Because the property you want — "no request from Ledger reaches an internal address" — is not a
property of one handler. It is a property of every line of code that opens an outbound connection,
and those are scattered.

- **The check and the connection are in different layers.** The handler validates a string; the
  standard library resolves and dials it later, and follows redirects by default. Any repair that
  lives only in the handler is repairing the wrong layer.
- **Every fetch is its own door.** The webhook test is not the only place Ledger calls out. It also
  *delivers* webhooks on the same tenant URL, and — the path that catches people — it fetches each
  tenant's branding logo while rendering `/v1/invoices/{id}/pdf`, the export route from the earlier
  chapters. Fix the test route alone and the logo fetch is still an open egress.
- **A blocklist is a guess about the internal network.** Blocking `169.254.169.254` misses the IPv6
  metadata address, the `metadata.google.internal` name, `127.0.0.1`, `10.0.0.0/8`, and whatever
  private range the network uses next year. The safe list is short and stable — public addresses —
  and everything else is denied.

The logo fetch is worth following, because it is where chapter 1's lesson comes back. There, the
repair for BOLA was not a check pasted into each handler but one loader, `LoadInvoiceFor`, that
every route had to pass through, so a new handler could not forget it. Egress needs the same shape.
The wrong fix is to copy the URL validation into the PDF renderer. The right fix is that Ledger has
exactly one way to make an outbound request — `egress` — and `http.Get`, `http.Client{}`, and their
kin are banned from handler code. The PDF renderer fetches the logo through the same client:

```go
func chapter07PDF(store *Store, fetcher outboundFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.LoadInvoiceFor(user, id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		logo := store.branding[invoice.Tenant].LogoURL
		if logo != "" {
			reply, err := fetcher.Fetch(r.Context(), logo)
			if err == nil {
				reply.Body.Close()
			}
		}
		renderViewPDF(w, publicViewFor(invoice)) // unavailable logo never exposes a row
	}
}
```

Now the property holds by construction. A reviewer does not have to check that each fetch validated
its URL; they have to check that each fetch went through `egress`, which is a `grep`. When Ledger
adds a fourth outbound call next quarter, it either uses the one client or it stands out in review
the way a bare `store.Invoice(id)` does now. Ops owns the egress the way it owns the gateway: one
place, one policy, and a record that these are the routes that reach outward.

The single choke point also fixes the second path for free — the logo fetch never had its own URL
check, and it does not need one, because it inherited the resolve-check-dial-no-redirect policy the
moment it stopped calling `http.Get`. That is the whole point of a boundary you cannot go around.

<!--mission-->
## Exercise: which of these reaches the metadata service?

Ledger's webhook test is `POST /v2/webhooks/test` with a body `{"url": "..."}`. A teammate proposes
this fix: "reject any URL whose host is `169.254.169.254` or `metadata.google.internal`." Below are
five requests, each with what DNS returns and what the destination replies. The public-address
answers are synthetic fixtures; no request is sent to those real IPs. For each: say what the
handler validates, what the socket would actually connect to, and mark whether the teammate's
blocklist stops the internal fetch and whether the chapter's `egress` policy stops it. Assume
vulnerable mode
otherwise follows redirects and dials whatever DNS returns. You do not need a running service.

```text
1. url = http://169.254.169.254/latest/meta-data/
   DNS: (literal IP)                 destination: 200, token page
2. url = https://hooks.cedar.example/ledger
   DNS: 8.8.8.8 (public)             destination: 200, "ok"
3. url = https://hooks.cedar.example/ledger
   DNS: 169.254.169.254              destination: no TLS; handshake fails
4. url = https://report.cedar.example/hook
   DNS: 1.1.1.1 (public)             destination: 302 -> http://169.254.169.254/
5. url = https://report.cedar.example/hook
   DNS: 1.1.1.1 (public)             destination: 200, "ok"
```

**Check your answer.**

- **1 is the decoy.** It is the one the blocklist was written for: host is the literal metadata IP,
  so the teammate's rule rejects it; the vulnerable handler's HTTPS rule rejects it too, and
  `egress` rejects it. If this were the only test, the blocklist would look like a fix. It is not;
  it is the case an attacker would never bother sending.
- **2 is safe and must stay working.** The handler validates `hooks.cedar.example`; the name
  resolves to `8.8.8.8`, a public address; the socket connects there and Cedar's webhook answers
  200. The blocklist allows it (correctly) and `egress` allows it (resolves to public, no
  redirect). This is the baseline: a fix that also breaks this has broken the feature.
- **3 exposes the missing boundary, and the blocklist misses it.** The handler validates
  `hooks.cedar.example` — not a blocked host — so the teammate's rule passes it. But DNS answers
  `169.254.169.254`, and the client dials it. The handshake fails, because the metadata service
  has no TLS, so no token comes back; what this case proves is a connection to an internal address
  that the caller chose. `egress` stops it because it checks the *resolved* address, not the
  name, and pins the dial to the address it approved. Trace: input `hooks.cedar.example` → name-based check passes → resolve to
  `169.254.169.254` → link-local check fails → refused before dial.
- **4 and 5 are the near-identical pair.** The two requests are byte-for-byte the same at
  submission: same URL, same public host, same DNS answer. The only difference is what the
  destination chooses to reply. In 5 it returns `200` and nothing happens. In 4 it returns a `302`
  to the metadata IP, and vulnerable mode follows it — the handler validated the first URL but
  fetched a second URL that was never validated. The blocklist misses 4 completely: it looked
  at the submitted host, which is innocent. `egress` stops 4 and allows 5 for the same reason —
  it does not follow the redirect — so under the fix both requests end at the public host and 4's
  internal fetch is stopped. The discriminator is redirect-following, decided after the initial
  URL check, which is why checking only that first URL cannot stop it.

If you marked 3 safe because the host was Cedar's own, that is the trust the screenshot server
extended to the seller's page: the name looked like ours, so the fetch was treated as ours.

*Incident from HackerOne report #341876, "SSRF in Exchange leads to ROOT access in all instances"
(0xacb, disclosed 23 May 2018), and Shopify, "One Million Dollars in Bug Bounties" (3 Apr 2019). The
method follows Colin Domoney, Defending APIs (Packt, 2024).*
