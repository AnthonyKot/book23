# The Row You Didn't Mean to Send

<!-- Incident claims are gated in checks/claims/03.tsv against David Lodge's Pen Test Partners disclosure and its archived JSON screenshot. The primary record proves the parcel-code map, derived postcode, tracking-page gate, session JSON and dated disclosure sequence. It does not establish the intended field policy for an authorized recipient, bulk enumeration or an affected-person count. -->

In September 2021 David Lodge of Pen Test Partners was looking at how DPD, the parcel carrier,
tells you where your delivery is. The tracking site had an API call that took a parcel code and
returned a small PNG: a map extract with the recipient's address highlighted. No login, no
postcode, just the code. A map is not an address. But the streets on it have names, and Lodge
found that searching a name like Verney Road, then checking which of the handful of matches also
had a Queen Catherine Road nearby, pinned the location down to a postcode.

The postcode mattered because it was the gate. DPD's tracking page for a parcel asked for the
recipient's postcode before it would show the delivery history, and the page had no login of its
own, only a transparent reCAPTCHA token. Parcel code plus postcode, and you were in. The page then
showed what you would expect: where the parcel was, when it would arrive.

The tracking page showed delivery information, but the session granted after the postcode check
also gave access to underlying JSON. Lodge found recipient contact and parcel details there. The
archived example includes a contact name and mobile field; its email field is null. The response
carried more than the delivery information shown on the page.

Lodge reported it on 2 September 2021; DPD asked for a proof of concept, received his response on
30 September, confirmed the problem on 1 October and confirmed resolution on 6 October. DPD then
asked that publication wait until the new year while it reviewed the rest of its estate. Pen Test
Partners called the disclosure process easy and clear.

That raises the property-level question in OWASP's **broken object property level authorization**,
number three on its API Security Top 10: once a request passes the object gate, which fields
should the server return? DPD's report proves a weak postcode gate and an underlying JSON
response with recipient contact details. It does not establish a separate field policy for an
authorized recipient. Ledger lets us isolate that second decision.

## Where the bug lives: the struct is the response

Chapter 1 left Ledger with `LoadInvoiceFor(user, id)` on invoice-by-ID routes and a tenant-filtered
list. Alice can no longer read invoice 205. What she reads when she asks for invoice 104 is the
question this chapter opens.

Here is the handler behind `GET /v2/invoices/{id}` after the chapter 1 fix. It is correct about
*which* invoice. Look at what it does with the invoice once it has it.

{{excerpt:ch03-vulnerable-read}}

`json.NewEncoder(w).Encode(inv)` serialises the stored row. The stored row is the struct Ledger
keeps in its store, and that struct has every field the business ever needed: the line items and
the amount, but also the customer's email and phone, the collections note Ledger's own finance
team writes when an invoice goes late, and the margin Ledger makes on the transaction. The web
app displays the amount, the status and the due date. It ignores the rest. The rest is still on
the wire, in every response, to every logged-in user of the tenant.

That is DPD's page: the client draws a subset; the server sends the row.

The fix is to stop encoding the stored type at all. Authorization decided which invoice Alice may
see. A second, separate decision says which *properties* of it she may see, and that decision
belongs on the server, in a type that exists for the purpose:

{{excerpt:ch03-view}}

Three details matter.

- **The view is a type, not a filter.** A blocklist ("strip `margin` before encoding") fails the
  day someone adds `discount_reason` to the stored struct. A view type lists what goes out; a new
  stored field is invisible until someone decides it should be seen.
- **The view is chosen after authorization, by the caller's role, not by the route.** Dana, Cedar's
  administrator, gets the customer's contact details; Alice does not. Both call the same route.
  `LoadInvoiceFor` decides *whether*; `ViewFor(user, inv)` decides *what*.
- **Internal fields have no view at all.** The collections note and the margin are Ledger's, not
  the tenant's. No customer-facing view carries them, so a route bound to a view cannot leak them
  by accident. DPD's page showed delivery information while its underlying JSON carried contact
  details; Ledger's view makes its own field policy explicit rather than relying on what the page
  displays.

## The same mistake, writing instead of reading

The read side sends the whole struct out. The write side takes the whole struct in. Here is
`PATCH /v2/invoices/{id}`, which lets a tenant set the purchase-order reference on an invoice:

{{excerpt:ch03-vulnerable-patch}}

`json.NewDecoder(r.Body).Decode(inv)` can write any matching exported field on the loaded
struct. The web app only ever sends `reference`. Alice can send `{"reference":"PO-9","status":"paid"}`
and mark invoice 104 paid, or `{"amount":0}`, or `{"tenant":"birch"}`. The loader checked that the
invoice was hers. The decoder let her rewrite it. OWASP folds this into the same category, under
the name **mass assignment**: the server bound client input to properties the client was never
meant to set.

The fix mirrors the read side. There is one decodable type for this route, and it has one field:

{{excerpt:ch03-patch}}

Go helps here more than most languages. `Decode` into `InvoicePatch{Reference string}` silently
drops `status` and `amount`, because they have nowhere to land. Turn on
`DisallowUnknownFields()` and it rejects them instead, which is the better answer: a client
sending `status` is either a bug or an attacker, and both should hear about it.

## If the fix is a type, why is it number three?

- **The struct is the schema, and the schema grows.** Every field the business needs ends up on
  the stored type, because that is where it is convenient. Nobody adds a field to the response
  on purpose; they add it to the struct, and the encoder does the rest. DPD's example JSON
  carried a mobile field beyond the delivery information shown on the tracking page.
- **The client hides it, so a page-only check misses it.** The web app shows amount, status and due
  date, and a reviewer who checks only the page sees exactly that. The network response still
  needs a separate check. Lodge found DPD's fields by reading the JSON, not the page.
- **Read and write are reviewed separately.** A team that has just built read views can still
  ship `Decode(inv)` on the next PATCH, because the review question "what does this return?" is
  not the same question as "what does this accept?".
- **The leak is symmetric with chapter 1, and hides behind its fix.** After `LoadInvoiceFor`, every
  invoice Alice sees is genuinely hers. That feels like the end of the problem. It is the end of
  the *object* problem. The property problem starts exactly there.

## The route you fixed, and the one you forgot

In the proposed fixed service, Ledger uses `ViewFor` on `GET /v2/invoices/{id}`, and the response
for invoice 104 loses its email, phone, note and margin. That will need a test. Two other routes
serve the same invoice.

`GET /v2/me/invoices` returns Alice's list. It was written after chapter 1, goes through the store
filter on Alice's tenants, and encodes `[]store.Invoice`, one row per invoice, margin and all. The
list endpoint leaks what the item endpoint now hides, and it leaks it for every invoice at once.

`GET /v1/invoices/{id}/pdf` renders the same row through a template. It goes through
`LoadInvoiceFor` since chapter 1, so it is the right invoice. The template was written when the
PDF was meant for Ledger's own finance team, and it prints the customer's email and phone in the
footer. The mobile app still calls it.

So the question after a property fix is the chapter 1 question again, one level down: not "which
routes reach this record?" but **"which routes encode this type, and does each one encode a
view?"** Ledger answers it the same way.

### 1. Find every encoder of the type

Grep for `Encode(` and for every template that binds `store.Invoice`. Ledger finds the item
route, the list route and the PDF template.

### 2. Make the stored type refuse to be encoded

Moving `ViewFor` into every handler is the chapter 1 mistake with a new name. Instead the stored
type gets a `MarshalJSON` that returns an error:

{{excerpt:ch03-marshal-guard}}

Once built, an attempt to JSON-encode the stored row returns "encode a view, not the row". The
compiler cannot enforce this one, but a test that checks the encoder error can catch the next
copied handler. The PDF template gets the same treatment: it is bound to `InvoiceView`, so a field
the view does not carry cannot be printed.

### 3. Test what comes out, not only what comes back

Chapter 1's loop asked, for every route, "does Alice get 404 for 205?". This chapter's loop asks,
for every route that returns 200, "which keys are in the body?". The table below is this
chapter's test, to run in both modes once its service code exists; in the vulnerable build the
assertions on the left hold, in the fixed build the ones on the right.

| Route | Caller | Request | Vulnerable build | Fixed build |
| :--- | :--- | :--- | :--- | :--- |
| `GET /v2/invoices/104` | Alice | — | body has `customer.email`, `margin` | body has `amount`; no `customer.email`, `collections_note`, `margin` |
| `GET /v2/invoices/104` | Dana | — | as above | body has `customer.email`; no `margin` |
| `GET /v2/me/invoices` | Alice | — | rows carry `margin` | rows lack `margin`, `customer.email` |
| `GET /v1/invoices/104/pdf` | Alice | — | PDF text contains the phone number | no phone, no email |
| `PATCH /v2/invoices/104` | Alice | `{"reference":"PO-9","status":"paid"}` | status becomes `paid` | reference updated, status unchanged |
| `PATCH /v2/users/me` | Alice | `{"is_admin":true}` | Alice is now an admin | field rejected, Alice unchanged |

The last row is the one to notice. Mass assignment on the user record is how a property bug
becomes an authorization bug: after that request, `ViewFor` faithfully gives Alice the admin view.

<!--mission-->
## Exercise: what leaves the server

Use Ledger as it stands after this chapter: `LoadInvoiceFor` on every invoice route, invoice 104
belonging to Cedar with amount £1,800.00, invoice 205 to Birch with £640.00, Alice an ordinary Cedar
user, Dana Cedar's administrator. Here are six handlers, as route → what the handler does:

```text
A. GET   /v2/invoices/{id}      -> inv := LoadInvoiceFor(user, id); Encode(ViewFor(user, inv))
B. GET   /v2/me/invoices        -> for inv in store.InvoicesWhere(tenant IN user.Tenants):
                                     out = append(out, ViewFor(user, inv)); Encode(out)
C. GET   /v1/invoices/{id}/pdf  -> inv := LoadInvoiceFor(user, id); render(pdfTemplate, inv)
D. PATCH /v2/invoices/{id}      -> inv := LoadInvoiceFor(user, id)
                                   var p InvoicePatch; Decode(&p); inv.Reference = p.Reference
E. PATCH /v2/invoices/{id}      -> inv := LoadInvoiceFor(user, id)
                                   var m map[string]any; Decode(&m)
                                   for k, v := range m { set(inv, k, v) }
F. PATCH /v2/users/me           -> u := currentUser(r); Decode(u); store.SaveUser(u)
```

For each one:

1. Say what type is encoded or decoded, and whether it is the stored row or a view.
2. Mark it safe or vulnerable.
3. For each vulnerable one, write one test: the caller, the request, and one key that must be
   absent from the response (or unchanged in the store) in the fixed build.

**Check your answer.**

- **A is safe.** The loader picks the invoice; `ViewFor` picks the properties by the caller's role.
  Test anyway: Alice fetches 104 and the body has no `margin`.
- **B is safe, though it looks like the list leak from the chapter.** It touches the store directly,
  and it encodes a slice, but every element went through `ViewFor` first. The question is not
  "does it encode a collection?" but "does each element pass through a view?". It does.
- **C is vulnerable.** The right invoice, through the loader, handed to a template bound to the
  stored row. Whatever the template prints, it can print. Test: Alice fetches the PDF for 104; the
  fixed build's PDF text contains no phone number. Fix: bind the template to `InvoiceView`.
- **D is safe.** `InvoicePatch` has one field, so `status` and `amount` in the body have nowhere to
  land. With `DisallowUnknownFields` they are rejected outright.
- **E is vulnerable, and it is D with the type removed.** Decoding into a map and setting each key
  is `Decode(inv)` written by hand: every key the client names becomes a property the client
  sets. Test: Alice sends `{"status":"paid"}` to 104; in the fixed build the status is unchanged.
  Fix: replace the map with `InvoicePatch`.
- **F is vulnerable, and it is the one that matters most.** The current user is loaded correctly
  and then overwritten by the body. `{"is_admin":true}` makes Alice an administrator, and from
  then on every view in this chapter gives her the admin's properties honestly. Test: Alice sends
  `{"is_admin":true}`; in the fixed build her record is unchanged and the request is rejected.
  Fix: decode a `ProfilePatch{DisplayName}`, nothing else.

If you marked E safe because the loader ran first, look again at what the loader decides. It
decides which invoice. Everything after it is a different question.

*Incident from David Lodge, "DPD package sniffing", Pen Test Partners, 7 February 2022. The method
follows Colin Domoney, Defending APIs (Packt, 2024).*
