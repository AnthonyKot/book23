# Who's Asking

<!-- claims to gate in checks/claims/01.tsv: 20 Jan 2021 private disclosure; POST /stats/workouts/details with editable ids; initially no authentication; 2 Feb 2021 silent partial fix requiring authentication; still readable by any registered member incl. private profiles; "3 million" members; journalist contact ~90 days; resolved within 7 days of CISO engagement; published 5 May 2021; data fields listed. Source: Jan Masters, Pen Test Partners, "Tour de Peloton: Exposed user data". -->

In January 2021 Jan Masters, a researcher at Pen Test Partners, was looking at the API behind
Peloton's bikes and app. One call stood out. For live-class details, the web app sent
`POST /stats/workouts/details` with a JSON list of IDs; the response exposed workout and personal
data, including data from profiles set to private. The app filled in the IDs. The researcher could
find IDs in the mobile app, change the request, and send it again.

At first the call needed no login at all. Masters reported it privately on 20 January. Peloton
acknowledged receipt and then went quiet. On 2 February a fix appeared without announcement: the
endpoint now required an authenticated session. It still returned whatever IDs you asked for.
Any of Peloton's three million members could pull the details of any other member, including
members who had set their profile to private. Masters' write-up puts it plainly: having a private
profile did not protect your data. Only after a journalist contacted the company, about ninety
days in, did Peloton's security team engage; the remaining issues were largely closed within a
week, and the report was published on 5 May.

The two versions of that endpoint are the whole subject of this chapter. The first asked nobody
who was calling. The second asked *who* was calling and never asked *whether that person may see
this record*. Those are different questions, and the second one is the one that gets skipped. OWASP
calls the skip **broken object-level authorization**, BOLA, and puts it first in its API Security
Top 10.

## Where the bug lives: an ID from the request, a lookup with no owner

Ledger's invoice endpoint has the same shape. Here is the handler behind `GET /v2/invoices/{id}`,
as it stands before this chapter's fix. `currentUser` is the login helper; `store` is the data
layer.

```go
func vulnerableInvoiceHandler(store *Store, pdf bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := currentUser(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		invoice, found := store.Invoice(id)
		if err != nil || !found {
			http.NotFound(w, r)
			return
		}
		writeInvoice(w, invoice, pdf)
	}
}
```

Read it the way Masters read Peloton's. `currentUser(r)` means the caller must be logged in, so the
route looks protected; that is Peloton's second version. But `id` comes straight from the URL, and
`store.Invoice(id)` fetches whatever row has that number. The handler checks login and discards
the returned user. Alice, logged in for Cedar, can walk the numbers: 104, 105, 106, and at 205 she is
reading Birch's invoice.

The fix adds the question the lookup forgot:

```go
func fixedInvoiceHandler(store *Store, pdf bool) http.HandlerFunc {
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
		writeInvoice(w, invoice, pdf)
	}
}
```

Three details matter.

- **The tenant comes from the stored invoice, not from the request.** If the check compared against
  a `tenant` field the client sent, Alice would send `cedar` with invoice 205 and pass.
- **A missing invoice and a forbidden invoice get the same answer.** Returning 404 for both means
  the endpoint cannot be used to learn which invoice numbers exist. A 403 is also defensible, if
  you accept that leak.
- **Unguessable IDs would not have fixed it.** A random ID only slows the walk until one leaks: in
  a URL, a log, an email, or another endpoint's response. Masters found IDs in Peloton's mobile
  app.

## If the fix is one line, why is it number one?

Because the line has to be in *every* handler that takes an object ID, and the way services are
built works against that.

- **Login happens once, in middleware.** Authentication, who is calling, is wired up globally and
  feels like security handled. Authorization, whether *this* caller may touch *this* record, has to
  happen per request, because only the handler knows which record is being asked for. Peloton's
  February change added login but left the per-record check missing.
- **Later steps trust earlier ones.** A web client may send IDs it has already displayed. If the
  server treats those client-supplied IDs as vetted, it skips the check only the server can enforce.
- **Ordinary tests pass.** A test where Alice reads invoice 104 passes with or without the check.
  Catching BOLA takes two users, and most test suites are written with one.
- **The attack is cheap.** Masters changed IDs in a request body. It required no special exploit
  chain and, after the partial fix, no account beyond one a user could register.

## The route you fixed, and the one you forgot

Here is how the fix goes wrong at Ledger. The service is invented; the shape of the failure is not.

A researcher reports that Alice can read invoice 205 at `/v2/invoices/205`. The team adds
`canReadInvoice`, writes a test proving Alice now gets 404 for 205, and ships. The report is
closed.

Three months later Birch's invoices turn up somewhere they should not. The v2 test still passes.
The leak came through `/v1/invoices/205`.

Nobody chose to leave v1 open. It was the original handler, and v2 began as a copy of it. v1 is
still deployed because an old version of the mobile app calls it, and the plan is to switch it off
once usage drops. It reads the same invoice table. The fix went into the v2 handler because that is
where the report pointed. v1 never got it.

Swapping `v2` for `v1` in a URL needs no tool. Neither does asking for the PDF: `/v1/invoices/205/pdf`
renders the same row through a route nobody on the current team knew was there. So the question
after a BOLA fix is not "is this handler fixed?" but **"which routes can reach this data, and does
every one of them check?"** Here is how Ledger answers it.

### 1. Find every door to the table

Three lists, compared. The routes the code registers: grep for `mux.Handle` and every path under
`/invoices`. The routes the gateway actually forwards. And the paths in live traffic that appear in
neither. At Ledger the first list turns up v1's read route and the PDF export. Lists and filters
count as doors too: `GET /v2/invoices?tenant=birch` reaches the same rows, so the rule has to apply
to the query, not only to the single-record lookup.

### 2. Move the check to where every route must pass

Patching v1 by hand repeats the original mistake: the protection lives in handlers, so the next
copied handler will miss it too. Instead, the one function that loads an invoice refuses to hand it
out without a user:

```go
func (s *Store) LoadInvoiceFor(user User, id int) (Invoice, bool) {
	invoice, ok := s.invoices[id]
	if !ok || invoice.Tenant != user.Tenant {
		return Invoice{}, false
	}
	return invoice, true
}
```

Every handler, in every version, calls `LoadInvoiceFor`. A bare `store.Invoice(id)` inside a route
handler becomes something review rejects on sight. Go can enforce it for you: move the store into
its own package and keep `Invoice` unexported, and the compiler does the review. Ledger's pilot
keeps store and handlers in one package, so there the rule is review's to keep.

### 3. Test every route with two users

The recipe for finding BOLA is the one Masters used: fetch a record as user A, confirm A can see
it, then ask for it as user B. Ledger turns that into a loop over every route that serves invoices.
These rows are covered in `service/ch01_test.go`, and the tests run in both modes: against the
vulnerable build they prove the leak happens; against the fixed build they prove it doesn't.

| Route | Caller | Invoice | Vulnerable build | Fixed build |
| :--- | :--- | :--- | :--- | :--- |
| `GET /v2/invoices/{id}` | Alice | 104 (Cedar) | 200, invoice | 200, invoice |
| `GET /v2/invoices/{id}` | Alice | 205 (Birch) | 200, Birch's invoice | 404, no data |
| `GET /v1/invoices/{id}` | Alice | 205 (Birch) | 200, Birch's invoice | 404, no data |
| `GET /v1/invoices/{id}/pdf` | Alice | 205 (Birch) | 200, Birch's PDF | 404, no PDF |
| `GET /v2/invoices/{id}` | Ben | 104 (Cedar) | 200, Cedar's invoice | 404, no data |

The loop is a list of routes, and a v3 nobody adds to it is a v3 the loop never visits. A separate
coverage assertion must compare registered invoice routes with the tested routes. Do not expect
monitoring to do this job. An alert on a spike in denied reads is worth having, but a route with
no check never denies anything.

### 4. Keep v1 behind the loader, and write down that it exists

v1 is not retired in this chapter. The mobile app still calls it, and "switch it off once usage
drops" is a plan, not a date. Until it has one, v1 goes through `LoadInvoiceFor` like everything
else, and Ops adds it to the inventory as a route that serves customer data. What "deprecated"
turns out to mean, and how a route that everyone has forgotten gets found and actually stopped, is
the subject of the chapter on zombie APIs.

<!--mission-->
## Exercise: find the open doors

Use Ledger as described: Alice belongs to Cedar, Ben to Birch, invoice 104 is Cedar's and 205 is
Birch's. You do not need a running service. Here are five handlers, written as route → what the
handler does:

```text
A. GET  /v2/invoices/{id}        -> LoadInvoiceFor(user, id)
B. GET  /v2/me/invoices          -> store.InvoicesWhere(tenant IN user.Tenants)
C. GET  /v1/invoices/{id}/pdf    -> renderPDF(store.Invoice(id))
D. POST /v2/refunds/quote        body: {invoice_id}
                                 -> LoadInvoiceFor(user, invoice_id); save quote q-771
   POST /v2/refunds/confirm      body: {quote_id, invoice_id}
                                 -> refund(store.Invoice(invoice_id))
E. GET  /v2/invoices?tenant=X    -> store.InvoicesWhere(tenant = X)
```

For each one:

1. Say where the object ID comes from, and whether anything compares it to the caller.
2. Mark it safe or vulnerable.
3. For each vulnerable one, write one two-user test: the caller, the request, and the expected
   result in the fixed build.

**Check your answer.**

- **A is safe.** The ID comes from the URL; `LoadInvoiceFor` compares the stored invoice's tenant to
  the caller's. Test it anyway: Alice requests 205 and gets 404 with no data.
- **B is safe, even though it queries the store directly.** Nothing in the request names an object.
  The filter is built from the logged-in user's own tenants, so Alice can only ever get Cedar's
  invoices. The question that matters is whether an ID comes from the caller, not whether the
  handler touches `store`. You could still route it through a shared `ListInvoicesFor(user)` so
  review has one rule to enforce, but that is tidiness, not a fix.
- **C is vulnerable. It is the forgotten v1 route.** The ID comes from the URL and goes straight to
  `store.Invoice`. Test: Alice requests `/v1/invoices/205/pdf`; fixed build returns 404, not
  Birch's PDF. Fix it with `LoadInvoiceFor`, then put it on the inventory.
- **D is vulnerable in the second step.** The quote step checks the invoice; the confirm step
  trusts a fresh `invoice_id` from the body. Test: Alice gets quote `q-771` for 104, then confirms
  with `invoice_id: 205`. Fixed build: rejected, and no refund on 205. Fix: confirm loads the quote,
  checks it belongs to Alice, and refunds the invoice stored with it, ignoring the body's
  `invoice_id`. This is Coinbase's bug in miniature: the step that validated and the step that
  acted looked at different fields.
- **E is vulnerable.** B and E look almost identical, but here the tenant comes from the query
  string and nothing compares it to the caller. Test: Alice requests `?tenant=birch`; fixed build
  returns an empty list or a denial, never Birch's invoices. Fix: filter on the tenants Alice may
  read, not on whatever the caller names.

If you marked D safe because its first step checks, reread the opener. That was the check Coinbase
assumed had already happened.

*Incident from Jan Masters, "Tour de Peloton: Exposed user data", Pen Test Partners, 5 May 2021.
The method follows Colin Domoney, Defending APIs (Packt, 2024).*
