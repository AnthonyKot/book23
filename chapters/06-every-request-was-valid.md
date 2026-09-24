# Every Request Was Valid

<!-- Incident claims: checks/claims/06.tsv (archived FTC complaint, order and release). The broker account is alleged; defendants neither admitted nor denied it. -->

In January 2021 the United States, on the Federal Trade Commission's behalf, filed a complaint in
the Eastern District of New York against a Long Island ticket broker called Just In Time Tickets
and its owner. It was the first case brought under the BOTS Act, and nothing in the complaint is a
vulnerability. Nobody bypassed a login. Nobody read another customer's data. The allegation is
that a broker bought tickets, and bought far more of them than the rules allowed.

What the government described was an ordinary-looking purchase flow, repeated past its posted
limits many thousands of times. From January 2017, the complaint says, the broker ran a program
called Automatick, later one called Tixman.
The operator typed in which tickets he wanted and what he would pay; the bot searched
Ticketmaster's sites, reserved any seats that matched, and held them while the owner decided
which to buy, "at least until the reservation clock expired". It kept the card and account details
on file and typed them in itself. It answered CAPTCHAs. Between 2017 and the filing, the
complaint says, the defendants made more than 14,186 purchases and came away with more than
48,451 tickets, exceeding posted limits for events including Elton John concerts, and resold them
for more than $8.6 million.

Ticketmaster had controls. The complaint lists them: purchase limits per event, a block on multiple
same-day purchases from one IP address, and a check that looked at the buyer's name, billing
address, account, IP address and cookies, and, until around October 2018, the card number. Every
one of those keys on *who is buying*. So the broker multiplied who was buying: more than 436
accounts, each with its own email address, opened under more than 320 names, more than 280
addresses, more than 450 credit cards, and over 12,500 IP addresses bought from rotating proxy
services. The complaint's word for what those identities did is "evade": the limits were there,
and the buyer was never the same buyer twice. The defendants settled without
admitting or denying the allegations, under an $11.2 million judgment of which $1,642,658.96
was payable and the rest suspended. Across the three brokers charged that week the FTC counted
more than 150,000 tickets.

The complaint never uses the word API, and the flow it describes was a website. But the shape is
the one OWASP calls **unrestricted access to sensitive business flows**, API6: a flow the
business meant to be used a little, exposed in a way that lets one party use it a lot. Taken one
at a time, the requests look valid. The harm is the volume, and the limit that was meant to bound
the volume was attached to a thing the attacker could manufacture.

## The flow the UI limits and the API does not

Ledger has one flow like this. A tenant user can refund an invoice: `POST /v2/refunds/quote`
with an invoice ID and an amount returns a quote (Alice's first quote for invoice 104 is `q-771`),
and `POST /v2/refunds/confirm` with the quote ID sends the money back to the customer. Cedar's
terms say refunds are exceptional: a tenant may refund up to £1,000 a day on its own, and anything
more needs Dana, the tenant admin, to approve. The web app enforces that. Once the day's refunds
reach £1,000 for Cedar the confirm button greys out and a note tells Alice to ask Dana.

Here is the confirm handler behind that button, as it stands after the earlier chapters. The quote
was created with `LoadInvoiceFor`, so it can only exist for an invoice Alice's tenant owns, and
confirm refunds the invoice stored with the quote, never one named in the body.

```go
func vulnerableChapter06Confirm(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, quote, ok := chapter06ConfirmInput(w, r, store)
		if !ok {
			return // session, body and quote tenant were checked
		}
		refund, status := store.UnrestrictedRefund(quote.ID, user, clock())
		writeChapter06Decision(w, status, refund)
	}
}

func (s *Store) UnrestrictedRefund(id string, actor User, at time.Time) (Refund, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quote, ok := s.quotes[id]
	if !ok || quote.Tenant != actor.Tenant {
		return Refund{}, refundMissing
	}
	if quote.Status == refundConfirmed {
		refund, found := s.refundByQuoteLocked(id)
		if found {
			return refund, refundConfirmed
		}
		return Refund{}, refundInvalid
	}
	if quote.Status == refundNeedsApproval {
		return Refund{}, refundNeedsApproval
	}
	invoice, found := s.invoices[quote.InvoiceID]
	if !found || quote.Amount <= 0 || quote.Amount > invoice.Amount-invoice.Refunded {
		return Refund{}, refundInvalid
	}
	return s.recordRefundLocked(quote, actor, at), refundConfirmed
}
```

Read it looking for £1,000. It is not there. The handler is a thin wrapper; the checks live in
the store method under it, `UnrestrictedRefund`. It checks that the quote belongs to the caller's
tenant, that it has not already been confirmed, and that the amount does not exceed what is left
on the invoice. All three checks are right and all three are per request. Nothing asks what
the tenant has already refunded today. A script that requests a quote for £400 on invoice 104 and
confirms it, then requests another and confirms that, then a third, has refunded £1,200 by the
third confirm. Each quote was valid. Each confirm was valid. The button that would have stopped
the third one is in a browser the script never opened.

Notice what did *not* go wrong. The cross-tenant refund from the first chapter is closed:
`q-771` can only ever refund invoice 104. Confirming `q-771` twice does not refund twice: the
second call sees the quote is already confirmed and returns the original refund. That is
idempotency, and it is worth having, but it protects one quote from being confirmed twice. The
script never confirms a quote twice. It asks for a new one.

The fix puts the business rule where the business flow actually executes:

```go
func fixedChapter06Confirm(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, quote, ok := chapter06ConfirmInput(w, r, store)
		if !ok {
			return
		}
		refund, status := store.ConfirmRefund(quote.ID, user, clock())
		writeChapter06Decision(w, status, refund)
	}
}
```

Three details decide whether this is real.

- **The total is computed from stored refunds, inside the store, at the moment of confirm.** The
  handler does not carry a counter; it asks `ConfirmRefund` to record the refund only if today's
  total for the tenant, plus this amount, stays under the allowance. Computing the total in one
  place and recording in the same call is what makes two confirms landing together add up
  correctly.
- **The allowance keys on the tenant, not the caller.** Cedar's money leaves Cedar's account
  whether Alice, Dana, or a script with Cedar's integration key presses confirm. A limit keyed on
  who is asking is the Ticketmaster limit, and the next section shows why.
- **Over the allowance is not an error, it is a different flow.** The response says so:
  `refund_needs_approval`, with the quote held for Dana. The script gets nothing; Alice with a
  genuine reason gets a path.

## Why the limit belongs on the object, not the identity

The obvious local fix is a counter per user: track what Alice has refunded today, refuse at
£1,000. It is quick, it makes the three-quote test pass, and it repeats the mistake in the
complaint.

Recall what Ledger established in the authentication chapter: a session token identifies a person;
the tenant key `ck_cedar_…` identifies Cedar's integration, not a person. The refund routes accept
both, because Cedar's accounting system issues refunds automatically. Here is the per-user
counter as it would sit in the handler:

```go
func refundedToday(user User, refunds []Refund, at time.Time) int {
	total := 0
	today := at.UTC().Format("2006-01-02")
	for _, refund := range refunds {
		if refund.Actor == user.Name && refund.At.UTC().Format("2006-01-02") == today {
			total += refund.Amount
		}
	}
	return total
}
```

Now run the same script through the integration key. The key resolves to a service identity that
has never refunded anything and gets a fresh bucket. Alice's session is at £1,000 and stopped;
Cedar's key confirms the fourth quote. Dana has her own bucket too. The limit was
written "per person" because a person was the only caller the author pictured, and the flow
accepts three kinds.

That is the Ticketmaster shape exactly. Their limit was per account, per card, per IP, and the
broker's whole method was to be many accounts, many cards, many IPs. A limit keyed on an identity
the caller can multiply is a limit on how many identities the caller can afford. A limit keyed on
the thing the business cares about, tickets per event or pounds per tenant per day, has nothing
to multiply.

Two more reasons this class is hard, both in the complaint.

- **The flow is not the bug.** Every check that belongs to a single request is present. The rule
  that was broken is a rule about a *sequence* of requests, and a handler holding one request has
  nowhere to put it.
- **The controls that exist look like security.** Ticketmaster had CAPTCHAs, IP blocks and card
  matching; the bot solved the CAPTCHAs and the proxies rotated the IPs. Ledger's per-route
  limiter, 30 requests a minute on lookup routes, would not notice three refunds in a morning.
  Rate limits bound how fast; this rule bounds how much.

## Where the limit lives, and what watches it

The fixed handler above calls into the store. Here is the store method, which is the choke point
for this chapter in the way `LoadInvoiceFor` was for the first.

```go
func (s *Store) ConfirmRefund(id string, actor User, at time.Time) (Refund, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	quote, ok := s.quotes[id]
	if !ok || quote.Tenant != actor.Tenant {
		return Refund{}, refundMissing
	}
	if quote.Status == refundConfirmed {
		refund, found := s.refundByQuoteLocked(id)
		if found {
			return refund, refundConfirmed
		}
		return Refund{}, refundInvalid
	}
	if quote.Status == refundNeedsApproval {
		return Refund{}, refundNeedsApproval
	}
	invoice, found := s.invoices[quote.InvoiceID]
	if !found || quote.Amount <= 0 || quote.Amount > invoice.Amount-invoice.Refunded {
		return Refund{}, refundInvalid
	}
	today := at.UTC().Format("2006-01-02")
	total := 0
	for _, refund := range s.refunds {
		if refund.Tenant == quote.Tenant && refund.At.UTC().Format("2006-01-02") == today {
			total += refund.Amount
		}
	}
	if total+quote.Amount > s.allowances[quote.Tenant] {
		quote.Status = refundNeedsApproval
		s.quotes[id] = quote
		return Refund{}, refundNeedsApproval
	}
	return s.recordRefundLocked(quote, actor, at), refundConfirmed
}
```

Every route that can move money out of a tenant, `/v2/refunds/confirm` and any batch or
integration route added later, goes through `ConfirmRefund`, which takes the tenant from the
stored quote. A handler cannot forget the allowance because the handler does not enforce it.

Three things sit around that method.

1. **The allowance is data, not a constant.** £1,000 is Cedar's figure, held on the tenant record
   and changeable by Ops. A limit buried in a handler cannot be raised for one tenant without a
   deploy, so eventually it gets removed instead.
2. **Exceeding it opens the admin flow.** The quote is parked with `status: needs_approval`, Dana
   sees it, and `POST /v2/refunds/{quote}/approve` is an admin-only route in the sense the
   function-level chapter established: the route declares its role, and Alice calling it for a
   Cedar quote gets 403 once the tenant-scope check has passed.
3. **Velocity is a signal, not a limit.** Ops records quotes per tenant per hour. A refund a
   minute for an hour, all under £1,000, is not blocked, it is flagged, because a rule about
   pounds cannot see a pattern about pace. The complaint's evidence was patterns: 436 accounts,
   12,500 addresses. Somebody has to be counting.

This table is the chapter's test, and it runs in both modes. The rows are sequences, not single requests, because the bug is a sequence.

| Sequence | Caller | Vulnerable build | Fixed build |
| :--- | :--- | :--- | :--- |
| quote £400 on 104, confirm; repeat three times | Alice (session) | three refunds, £1,200 out | two refunds; third confirm returns `refund_needs_approval`, £800 out |
| confirm `q-771` (£400) twice | Alice (session) | one refund, second returns the same refund | same: one refund, £400 out |
| after Alice reaches £1,000, quote and confirm £400 | `ck_cedar_…` (integration) | refund goes through | `refund_needs_approval`; Cedar total stays £1,000 |
| after Alice reaches £1,000, quote and confirm £400 | Dana (session) | refund goes through | `refund_needs_approval`; Dana approves it through the admin route, £1,400 out |
| quote on 104, confirm | Ben (session) | 404 on quote | 404 on quote |

The last row is the first chapter's test, kept in the loop so nobody trades one for the other.

<!--mission-->
## Exercise: which of these is the volume bug?

Ledger as described: Alice and Dana are Cedar's, Ben is Birch's, invoice 104 is Cedar's for
£1,800.00, invoice 205 is Birch's for £640.00. Cedar's daily refund allowance is £1,000. Five
handler shapes, written as route → what the handler does:

```text
A. POST /v2/refunds/confirm      body: {quote_id}
                                 -> q := quote(quote_id); if q.Confirmed { return q.Refund }
                                    refund(q); mark confirmed
B. POST /v2/refunds/confirm      body: {quote_id}
                                 -> q := quote(quote_id); if refundedToday(user) + q.Amount > allowance
                                    { return needs_approval }; refund(q)
C. POST /v2/refunds/confirm      body: {quote_id}
                                 -> q := quote(quote_id); store.ConfirmRefund(q.Tenant, q)
D. POST /v2/refunds/quote        body: {invoice_id, amount}
                                 -> inv := LoadInvoiceFor(user, invoice_id);
                                    if amount > inv.Amount - inv.Refunded { 400 }; save quote
E. POST /v2/invoices/{id}/remind -> LoadInvoiceFor(user, id); email the customer a reminder
```

For each: name the input or event the handler is reacting to, the check that bounds the *sequence*
(if any), and the state after the sequence in its answer. A uses three fresh £400 quotes on invoice
104. B and C start after Cedar has refunded £1,000 today and try one more £400 quote through
`ck_cedar_…`. D creates fifty £400 quotes without confirming them; E sends fifty reminders.

**Check your answer.**

- **A is the decoy.** It looks defended: a repeated confirm of `q-771` returns the original refund
  and moves no money. Trace the script, though. The third request uses a third fresh quote, each
  unconfirmed and valid against invoice 104's remaining balance. Check that runs: "already
  confirmed?" (no). State after three: £1,200 refunded, past Cedar's £1,000 allowance.
  Idempotency bounds *one quote*; nothing bounds the flow.
- **B and C are the pair.** They differ in what the total is keyed on. With Cedar already at
  £1,000, run one £400 quote through `ck_cedar_…`. In B, `refundedToday(user)` looks up a service
  identity that has refunded nothing, so the check passes and Cedar reaches £1,400. In C,
  `ConfirmRefund` totals Cedar's refunds regardless of who confirmed them, so it returns
  `refund_needs_approval` and Cedar stays at £1,000. C is fixed; B repeats Ticketmaster's
  per-account limit.
- **D is safe for this class, and it is the one people mark vulnerable.** Fifty quotes for £400 on
  invoice 104: the check is per quote, against the invoice's remaining balance, and confirm
  rechecks it. But a quote moves no money. Fifty quotes is fifty rows in a table, not a business
  event. The money leaves at confirm, so confirm is where the allowance lives; a cap on open
  quotes per invoice is worth adding, but that is a resource rule, not this one.
- **E is vulnerable, and it is the same bug in a different flow.** Input: an invoice ID that Alice
  may read. Check that runs: `LoadInvoiceFor` (passes; it is hers). Check on the sequence: none.
  State after fifty: fifty emails to the customer of invoice 104 from Ledger's domain. Every one
  authorised. The web app shows one "remind" button per invoice per day; the route shows none.
  The fix is the fixed confirm handler's shape: a per-invoice, per-day allowance recorded in the
  store, and a different response when it is used up.

If you marked A safe because a duplicate confirm is harmless, you were reading one request. The
complaint is about fourteen thousand.

*Incident from* United States v. Just In Time Tickets, Inc. and Evan Kohanian, *complaint and
stipulated order, E.D.N.Y. 21-CV-215, January 2021, and the FTC's release of 22 January 2021.
The method follows Colin Domoney, Defending APIs (Packt, 2024).*
