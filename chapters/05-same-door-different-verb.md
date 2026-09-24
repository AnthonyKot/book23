# Same Door, Different Verb

<!-- Incident claims: checks/claims/05.tsv (archived primary slides and corroboration). Keep the automaker unnamed; API5 is the book's classification. -->

In January 2025 Eaton Zveare spent a weekend on the web portal a large carmaker runs for its
dealers across the United States. Dealers use it to order cars, record sales and manage customers.
It is invite-only, guarded by two-factor authentication, and built on an SAP/Java backend with an
AngularJS front end. It looked shut.

Getting an account was the first flaw, and it is not this chapter's subject: registration required
an invite token, but the server accepted the request whether the token was valid or not — a broken
authentication check, so anyone could register. Zveare said so himself afterwards, that "only two
simple API vulnerabilities blasted the doors open, and it's always related to authentication." The
account this gave him had no permissions at all. A profile-update action then established a usable
session. The interesting failure is what happened next.

The portal had an administrative user-management page. An ordinary account was not supposed to
reach it, and in the browser it did not: the front end checked the account's role and refused to
render the page. But that check lived only in the browser. When Zveare made the page render anyway
and its create-user function ran, the server took the request, read the elevated role the form
offered, and created the account. The result was a "national admin" with access across the
platform. A later impersonation feature let Zveare reach other dealer systems without their login
or two-factor checks. More than a thousand dealerships sat behind the admin account. Zveare's
slides record verified fixes on 11 February, eight days after his report; he told TechCrunch the
unnamed carmaker found no evidence of earlier exploitation.

The lesson is the one this chapter is about. The server authenticated the caller — it knew *who*
was asking. It never asked *whether this caller may run this function*. Those are different
questions, and the second is the one that gets left to the front end, where it protects nobody.
OWASP calls the skip **broken function-level authorization**, and it sits fifth in the API Security
Top 10.

## The check you already have answers a different question

Ledger has this shape too, and it is easy to miss because chapter 1 looks like it covered it. Every
invoice-by-ID route gets its invoice through `LoadInvoiceFor(user, id)`, which refuses to return a row
from another tenant. That answers *may this caller see this invoice*. It says nothing about *what
this caller may do to it*. Here is the delete route as it stands before this chapter's fix:

```go
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
```

Read what the check proves. `LoadInvoiceFor` confirms invoice 104 belongs to Alice's tenant, Cedar,
so Alice clears it — and then the row is deleted. The same function that keeps Alice out of Birch's
invoice 205 waves her through to *destroy* her own tenant's 104, a thing only a tenant admin should
do. Dana, Cedar's admin, and Alice, an ordinary Cedar user, reach this route with exactly the same
result, because the only question asked is one they both pass.

`GET /v2/invoices/104` and `DELETE /v2/invoices/104` are the same URL. The verb is the whole
difference between reading a record and removing it, and in the vulnerable build the two verbs share
one authorization story. Add the admin routes Ledger grew — `GET /v2/admin/users` lists a tenant's
people — and the gap widens: those routes were registered under an `/admin` path prefix, and the
name was mistaken for a guard.

## Why a check in the handler is not enough

The tempting fix is one line in the delete handler: `if user.Role != "tenant-admin" { 403 }`. It
works for this route. It is also chapter 1's original mistake, worn a second time.

- **The protection lives in the handler, so the next handler misses it.** The admin void route,
  `POST /v2/admin/invoices/{id}/void`, was copied from a working route and never got the line. A
  reviewer reading the delete handler sees the check and moves on; the void handler two files over
  has none, and nothing about reading either one reveals the other's gap.
- **A path prefix is not a permission.** `/v2/admin/...` looks privileged, but the string in the
  URL enforces nothing on its own. Worse, the rule "an admin prefix means admins only" was never
  even true across Ledger: `/v1` has no admin prefix and never did, so any reasoning that keys off
  the prefix is reasoning about a convention the older routes don't share.
- **The front end's check protects nobody.** This is the dealer portal exactly. Hiding the page in
  the browser stops the honest click and none of the dishonest requests. The server is the only
  place the decision can be enforced, because the server is the only place the attacker cannot edit.

So the fix is not "guard the delete route." It is "make every route state, in one place, who may
call it, so that a route with no statement fails loudly instead of defaulting open."

## Declare the access every route needs, and let a grep prove it

Ledger already lists its routes in one table. This chapter gives each entry an access level and
refuses to register a route without one:

```go
var chapter05Declarations = []Route{
	{Method: http.MethodGet, Pattern: "/v2/invoices/{id}", Access: UserAccess},
	{Method: http.MethodPatch, Pattern: "/v2/invoices/{id}", Access: UserAccess},
	{Method: http.MethodDelete, Pattern: "/v2/invoices/{id}", Access: TenantAdmin},
	{Method: http.MethodGet, Pattern: "/v2/admin/users", Access: TenantAdmin},
	{Method: http.MethodPatch, Pattern: "/v2/admin/users/{id}", Access: TenantAdmin},
	{Method: http.MethodPost, Pattern: "/v2/admin/invoices/{id}/void", Access: TenantAdmin},
	{Method: http.MethodGet, Pattern: "/v1/invoices/{id}", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v1/invoices/{id}/pdf", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v2/me/invoices", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v2/invoices", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/refunds/quote", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/refunds/confirm", Access: UserAccess},
	{Method: http.MethodGet, Pattern: "/v1/invoices", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/auth/login", Access: Public},
	{Method: http.MethodGet, Pattern: "/v2/me", Access: UserAccess},
	{Method: http.MethodPatch, Pattern: "/v2/users/me", Access: UserAccess},
	{Method: http.MethodPost, Pattern: "/v2/auth/otp/request", Access: Public},
	{Method: http.MethodPost, Pattern: "/v2/auth/otp/verify", Access: Public},
}
```

`Access` has no usable zero value: a route that forgets to set it does not quietly become `Public`,
it fails to register, and a test walks the table and asserts every route declares a level. The
levels are `Public`, `User` and `TenantAdmin` (in the code the middle one is spelled
`UserAccess`, because `User` is already the name of a type). After `currentUser`, an invoice-by-ID route runs
`LoadInvoiceFor` first, returning 404 for an out-of-tenant ID; access middleware then checks the
role before the handler:

```go
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
```

Now the delete handler does no role logic of its own; it is reached only after the invoice scope
and the route's declared level admit the caller. Tenant scoping stays in `LoadInvoiceFor`; the role
decision is the middleware's. `grep "Access:" service/routes.go` lists the declarations, while
registration and the route-table test catch a missing one before the service serves requests.

One deliberate difference from chapter 1: when the middleware denies Alice the delete, it answers
**403**, not the **404** Ledger returns for a cross-tenant read. In chapter 1 the 404 hid whether
the invoice existed. Here it is Alice's own tenant's invoice; she can already see it through `GET`.
There is no existence to hide, so the honest answer is "you may not do this," and 403 says it.

## The door you shut in chapter 3, reached through another route

Chapter 3 closed self-promotion: `PATCH /v2/users/me` stopped decoding arbitrary fields and took a
typed `ProfilePatch{DisplayName}`, so Alice could no longer set `role` on herself. That repair holds
for that route.

But the admin surface added its own way in. `PATCH /v2/admin/users/{id}` lets a tenant admin change
another Cedar user's role, and in the vulnerable build it is reachable by any logged-in user,
because it lives behind the `/admin` prefix and nothing checks the caller's level. Alice sends
`{"role":"tenant-admin"}` for her *own* user ID and becomes an admin — the exact door chapter 3
shut, standing open again one route over. And once Alice is an admin, chapter 3's `ViewFor` honestly
starts handing her the admin view, customer contact details included: a function-level gap turning
into a data-level one. The declared-access fix closes it the same way it closes the delete: the
route is declared `TenantAdmin`, the middleware denies Alice with a 403, and her record is unchanged.

The pair to hold in mind: `PATCH /v2/users/me` (typed, level `User`, safe) and
`PATCH /v2/admin/users/{id}` (level `TenantAdmin`, denied to Alice). Same actor, almost the same
request, opposite outcomes — and the thing that separates them is which route's declaration the
caller had to clear.

<!--mission-->
## Exercise: which verb, which caller, which door

Ledger as before: Alice is an ordinary Cedar user, Dana is Cedar's tenant admin, Ben is a Birch
user; invoice 104 is Cedar's and 205 is Birch's. Each route below carries the same declaration in
both builds; in the vulnerable build the `TenantAdmin` level is declared and not enforced. You do not need a running
service. For each case, name the caller's role, say which route declaration decides
the request, and give the response and whether any row changed.

```text
A. PATCH  /v2/invoices/104        Alice   {"reference":"PO-9"}      (level: User)
B. DELETE /v2/invoices/104        Alice   —                        (level: TenantAdmin)
B'. DELETE /v2/invoices/104       Dana    —                        (level: TenantAdmin)
C. DELETE /v2/invoices/205        Dana    —                        (level: TenantAdmin)
D. GET    /v2/admin/users         Alice   —                        (level: TenantAdmin)
D'. GET   /v2/admin/users         Dana    —                        (level: TenantAdmin)
E. PATCH  /v2/admin/users/{alice} Alice   {"role":"tenant-admin"}  (level: TenantAdmin)
E'. PATCH /v2/users/me            Alice   {"role":"tenant-admin"}  (level: User)
F. GET    /v2/Admin/users         Alice   —                        (note the capital A)
```

**Check your answer.**

- **A is safe, and it is the decoy.** Editing a reference is an ordinary user's job, so the route is
  declared `User`; Alice's role clears it. Vulnerable and fixed builds both return 200 and set the
  reference. A write is not the same as a privileged write — do not deny it just because it changes
  something.
- **B is the bug.** Alice's role is `User`; the delete route needs `TenantAdmin`. Vulnerable build:
  `LoadInvoiceFor` passes on her own tenant's 104 and the row is deleted (204). Fixed build: the
  middleware denies her before the handler runs — 403, and 104 is still there.
- **B' is the same route with the right caller.** Dana is `TenantAdmin`, so she clears the
  declaration; 204 and 104 is gone, in both builds. B and B' are the pair: identical request, the
  caller's role is the whole difference.
- **C never reaches the role check.** Dana is a Cedar admin; 205 is Birch's. `LoadInvoiceFor` returns
  not-found first, so both builds answer 404. Tenant scope is checked before role, and being an
  admin of your own tenant is not being an admin of someone else's.
- **D is vulnerable.** Alice is `User`; the user-list route needs `TenantAdmin`. Vulnerable build:
  the declaration is not enforced, so any logged-in caller gets Cedar's user list (200). Fixed build: 403. This is the
  dealer portal's admin page — hidden in the front end, open at the API.
- **D' is D with the right caller.** Dana clears `TenantAdmin`; 200 in both builds.
- **E is the second door, and it undoes chapter 3.** Alice is `User`; the admin route needs
  `TenantAdmin`. Vulnerable build: her request is honoured, `role` becomes `tenant-admin`, and from
  then on `GET /v2/invoices/104` shows her the admin view with customer email. Fixed build: 403, her
  record unchanged. Trace it: caller's role → the route's declaration → denied before the body is
  ever decoded.
- **E' is the pair, and it is safe.** `PATCH /v2/users/me` is level `User`, but chapter 3 made it
  decode a typed `ProfilePatch{DisplayName}`, so `role` is not a field it can set. Rejected as an
  unknown field in both builds. E and E' show that shutting one route does not shut the function:
  the admin route needed its own declaration.
- **F is the trap.** `/v2/Admin/users` with a capital A is not the registered route, so Go's mux
  returns 404 in both builds. It never reaches a handler or a role decision. A request for the real
  path reaches the route and is decided by its declaration in the fixed build. The path is not the
  permission; the declaration is.

*Incident from Eaton Zveare, "Unexpected Connections," DEF CON 33, 10 August 2025, corroborated
by Zack Whittaker, TechCrunch, 10 August 2025. The method follows Colin Domoney, Defending APIs
(Packt, 2024).*
