# Where Every Route Must Pass

<!-- No incident in this chapter. The route-wide properties and exercise are asserted by service/ch11_test.go; the two Go blocks are marked excerpts. -->

The opener asked one question of a Coinbase trade: which check did this request assume had
already happened? Ten chapters gave ten answers, and each answer was a place as much as a check.
The loader that asks whose invoice. The session store that says who is asking. The view type
that decides which fields leave. The counter keyed on the thing being guessed. The declaration
that says which role a route needs. The store method that decides a refund under one lock. The
one client every outbound request goes through. The settings literal a test reads. The inventory
that reconciles the hosts. The rate that is stored once and read forever after.

Read as a list, those are ten habits. Read as code, they are one road. Every request that
reaches Ledger takes it, and each chapter's check stands at a fixed point on it. This chapter
walks the road once, end to end, and then shows the one test that walks it for you.

## Every request takes the same road

Here is the finished build, the one every chapter's fixed side accumulates into, composed in one
place. Go wraps handlers from the inside out, so the code reads bottom-up: the last line is the
first thing a request meets.

```go
func chapter11RequestPath(mux *http.ServeMux, routes []Route, store *Store,
	sessions *sessionStore, limits *chapter04Limits, settings Settings) http.Handler {
	var handler http.Handler = mux
	handler = chapter09LookupGateway(Fixed, limits, handler)
	handler = chapter05Authorization(Fixed, mux, routes, store, sessions, handler)
	handler = chapter08ErrorWriter(settings, store, handler)
	handler = chapter02Identity(Fixed, sessions, handler)
	handler = chapter09Gateway(Fixed, handler)
	return handler
}
```

Read it in the order a request does.

1. **The host filter** (the inventory chapter). `r.Host` must be `api.ledger.example`. Any
   other name, including `ledger-staging.internal`, gets 404 before a token is read. A host Ops
   did not declare is not a host Ledger serves.
2. **Identity** (the authentication chapter). The bearer token is looked up in the session store.
   If it is a live session, the request now carries a person; if it is a tenant key, it carries a
   service account; if it is the mobile app key or nothing, it carries nobody. This layer refuses
   nothing. It only decides who the rest of the road is talking to.
3. **The error writer** (the misconfiguration chapter). It sits outside authorization on purpose.
   Whatever a lower layer says about a `GET /v2/invoices/{id}` that failed, the body that leaves
   production is `{"error":"not found"}`, because `Debug` is false in the literal this build was
   constructed from.
4. **Authorization** (the object-level and function-level chapters, in that order). For a route
   that names an invoice, `LoadInvoiceFor` runs first and answers 404 if the invoice is not the
   caller's; only then does the declared access level run and answer 403 if the caller's role is
   short. Tenant scope before role, so a request can never learn that a foreign invoice exists by
   being told it lacks the role to touch it.
5. **The lookup budget** (the resource chapter). Thirty a minute per tenant and route on the
   email lookups. It sits inside identity, so the key is the tenant, not the address, and a
   thousand rented addresses share one counter.
6. **The handler.** Only now does the code a chapter printed run: the loader for the invoice, the
   view for the response, the typed patch for the body, the store's `ConfirmRefund` for the
   money, `egress` for anything that leaves the building, the stored `FXRate` for anything in
   euros.

Six layers, and not every chapter's repair is on them. The ones that are not have a place of
their own, and that is the next section.

## What the order buys you

The layers are not a pile. Swap two and a chapter's repair quietly stops working, while every
test that checks one layer alone keeps passing.

- **Identity before the budget.** Put the lookup counter outside identity and it can only key on
  what it has, which is the client address. That is the Instagram limiter: real, busy, and
  counting the wrong thing. The counter needs the tenant, so it must stand after the layer that
  knows one.
- **Scope before role.** Put the role check first and Dana, a Cedar admin, sends
  `DELETE /v2/invoices/205` and gets 403: she has learned that 205 exists and that a role could
  delete it. With the loader first she gets 404, the same answer a random number gets. The
  function-level chapter's declaration table is only safe because the object-level chapter's
  loader runs ahead of it.
- **The error writer outside authorization.** Put it inside and it wraps the handler only; a 404
  raised by the scope layer would go out with whatever body that layer wrote. Outside, it sees
  every 404 on `GET /v2/invoices/{id}` after the host filter passes, regardless of which lower
  layer raised it, and the body is one string.
- **The host filter outside everything.** Every other layer is keyed on something the request
  carries, a token, a path, a tenant. The host filter is keyed on the list Ops wrote down. If it
  sat inside identity, a staging request would be given a person before being refused, and the
  session store's behaviour would become part of what the staging host exposes.

None of those orderings is visible in a handler. They are visible only in the composition, which
is why the composition is printed here and nowhere else.

## The controls that are not on the road

Three of the book's repairs never see a request, one acts on the way out rather than the way in,
and one has no request to see. If you look for them on the road you will conclude they are
missing.

- **The declaration table is checked at construction.** Registering a route without an access
  level panics before the server listens. The function-level chapter's test is not a request; it
  is the fact that the process started.
- **The settings literal is checked at construction.** `validateProductionSettings` walks the
  route table against `PublicRoutes` and refuses `Debug: true`, a wildcard origin, or a public
  route the allow-list does not name. The misconfiguration chapter's exposure, a route declared
  `Public` by copy-paste, cannot reach a running production build.
- **The inventory is a test.** `ReconcileInventory` compares declared hosts, code routes, gateway
  surfaces and traffic, and the fixed build's output is empty. It runs in the build, not in the
  request, because the host it is looking for is one no request would tell you about.
- **`egress` acts on the way out.** The webhook test passes every inbound layer as an ordinary
  user's request, and only then does the outbound client resolve, refuse the private address, or
  decline the redirect. The earlier PDF logo route used the same client before its `/v1` route
  was retired. A 400 from `egress` is a request that was perfectly authorized.
- **The rate feed has no inbound request at all.** `PollRates` is the hourly fetch; nothing in
  the fixture schedules it, and nobody calls in. Its input is the partner's response, and its checks, the
  strict row, the band, the last good rate, are the road's checks pointed the other way. The
  stored `FXRate` is what lets the refund quote on the road stay ignorant of the feed.

So "where every route must pass" is three places, not one: the composition above for the way in,
`egress` for the way out, and construction time for the shape of the service itself. A control
in any other place is a control on the route someone happened to test.

## The test that reads the whole table

The object-level chapter closed with a loop: for every invoice route, request Cedar's invoice as
Alice and as Ben. The finished build has twenty routes, three callers with different roles, and
a fourth caller with no token. The loop grows to match, and it reads the route table rather than
a list someone typed.

```go
func TestChapter11EveryRouteEveryCaller(t *testing.T) {
	routes := NewChapter10App(Fixed).Routes()
	if len(routes) != 20 {
		t.Fatalf("route inventory changed: %d", len(routes))
	}
	for _, route := range routes {
		for _, cell := range chapter11ConcreteRequests(t, route) {
			for _, caller := range chapter11Callers {
				t.Run(route.Method+" "+cell.path+"/"+caller.name, func(t *testing.T) {
					app := chapter11App()
					path := cell.path
					if route.Pattern == "/v2/invoices" && caller.tenant != "" {
						path += "?tenant=" + caller.tenant
					}
					got := chapter04Request(t, app, route.Method, path, caller.token, "192.0.2.44", cell.body)
					chapter11AssertTenant(t, caller, got)
					chapter11AssertNoInternalFields(t, got)
					chapter11AssertOpaqueInvoice404(t, route, got)
					chapter11AssertDeclaredStatus(t, route, cell, caller, got)
				})
			}
		}
	}
}
```

Four properties, checked for every registered route and every caller, on the fixed build:

- **No body ever carries another tenant's data.** Ben's responses never contain the string
  `Cedar`, invoice 104's number, or Alice's name; Alice's never contain Birch's. This is the
  object-level chapter's promise made for routes that chapter never saw.
- **No body ever carries an internal field.** `CollectionsNote` and `Margin` appear in no
  response to anyone, Dana included. The property chapter's view is in force on every route that
  encodes an invoice, including the ones added after it.
- **Every 404 on `GET /v2/invoices/{id}` at the public host has the same body.** Whichever lower
  layer raised it.
- **A refused status comes from the layer the declaration predicts.** Anonymous on a `User`
  route is 401; a user on a `TenantAdmin` route is 403; anyone on a foreign invoice is 404; and a
  `Public` route does not refuse a request merely for having no token. A valid request may
  answer 200; an incomplete one may still get 400 from its handler.

What the loop does not check is as important. It sends one request per cell, so it cannot see the
business-flow chapter's third refund or the resource chapter's thirty-first lookup; those are
sequence tests in their own files. It runs in one process, so it cannot see a host that neither
the gateway nor the logs mention; that is the inventory's limit, stated there. And it cannot see
Coinbase's bug. The trader was logged in, spending his own money, on a route that answered him
correctly; every cell in a table like this one would have been green. The check that was missing
was "is this the object the previous step approved?", and Ledger's version of it lives in two
handlers that the loop only touches once: confirm refunds the invoice stored with the quote, and
the euro quote converts at the rate stored with the invoice. Those are asserted as sequences too.

Which is the last answer to the opener's question. The check a request assumes has already
happened is, in a service built this way, one of three things: a layer on the road that runs
before the handler, a rule enforced when the process starts, or a fact recorded by an earlier
step and read back instead of recomputed. When you find a check that is none of those, you have
found the next chapter of somebody's incident report.

<!--mission-->
## Exercise: which layer answers?

Use the finished fixed build. Alice is a Cedar user, Dana is Cedar's tenant admin, Ben is a Birch
user; invoice 104 is Cedar's, 205 is Birch's. For each request, name the layer that produces the
response (host filter, identity, error writer, scope, role, budget, handler, `egress`, or
construction time) and the status. You do not need a running service.

```text
A. anonymous   GET    api.ledger.example/v2/health
B. anonymous   GET    api.ledger.example/v2/admin/health
C. Alice       GET    api.ledger.example/v2/invoices/205
D. Alice       DELETE api.ledger.example/v2/invoices/104
E. Dana        DELETE api.ledger.example/v2/invoices/205
F. Alice       GET    api.ledger.example/v2/invoices?email=x@cedar.example   (31st in a minute)
G. Alice       GET    ledger-staging.internal/v2/invoices/104
H. Alice       POST   api.ledger.example/v2/webhooks/test  {"url":"https://hooks.cedar.example/ledger"}
                      (the name resolves to 169.254.169.254)
```

**Check your answer.**

- **A: handler, 200.** `GET /v2/health` is declared `Public` and named in `PublicRoutes`, so the
  role layer passes a request with nobody in it and the handler writes `{"ok":true}`. No layer
  refuses it, and that is the point of the allow-list: the routes that answer strangers are four,
  written down, and checked against the table when the process starts.
- **B: role, 401.** One path segment away from A, opposite outcome. The admin probe is declared
  `TenantAdmin`; the role layer finds no user and answers 401 before the handler runs. Dana would
  get 200 and a body with a version and nothing else. In the vulnerable build this route was
  `Public` by copy-paste, and the settings validator is what refuses that build now.
- **C: scope, 404, and it is the decoy.** The status looks like a route that does not exist. It
  is the loader: `LoadInvoiceFor(alice, 205)` sees Birch on the row and returns not-found, the
  scope layer answers 404, and the error writer replaces the body with `{"error":"not found"}`.
  A random invoice number gets the identical bytes. If you named the host filter or the router,
  reread the road: the host is right and the route is registered. The answer came from the
  object-level chapter.
- **D: role, 403.** Alice's own invoice, so scope passes and hands the row on. The delete route is
  declared `TenantAdmin`, Alice's role is `user`, and the role layer answers 403. Nothing is
  deleted. 104 is still there for Dana to delete.
- **E: scope, 404.** D with the caller and the invoice swapped, and the pair to D. Dana has the
  role. She never gets to use it: 205 is Birch's, the loader returns not-found, and scope answers
  404 before the role layer runs. If you gave E a 403 you have put role before scope, and told
  Dana that 205 exists.
- **F: budget, 429.** Alice is authenticated and authorized; the request would succeed. The
  lookup gateway counts it against `(Cedar, /v2/invoices)` for this minute, finds thirty already,
  and answers 429. Send it from a different address and the count is the same, because the key is
  the tenant identity got the layer above.
- **G: host filter, 404.** Alice's token is valid and 104 is hers, and none of that is read. The
  host is not `api.ledger.example`, so the first layer answers 404 and the request never reaches
  identity. Before the inventory chapter's fix this host forwarded `/v1/` paths; now it forwards
  nothing, and the same request on the public host would be a 200.
- **H: `egress`, 400.** Every inbound layer passes: right host, live session, `User` route, no
  invoice to scope. The handler calls `egress.Fetch`, which resolves the name, finds a link-local
  address in the answer, and refuses before dialling. The status is a 400 from a request that
  was completely authorized, which is why the outbound client is a layer of its own and not a
  check in a handler.

If you named the handler for C, or the role layer for E, you have found the reason this chapter
prints the composition rather than another handler. Handlers do not know what ran before them.
The road does.

*No incident. Ledger's composition and the every-route loop are in `service/`; the method
throughout follows Colin Domoney, Defending APIs (Packt, 2024).*
