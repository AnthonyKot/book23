# One Key for Every Door

<!-- Incident claims and source locators: checks/claims/02.tsv. -->

In September 2021 Alan Monie of Pen Test Partners examined code in the Android app of BrewDog, the
Scottish brewer whose customers include more than 200,000 "Equity for Punks" shareholders. In the
source he found a bearer token used with three of the brewer's API endpoints: one key written
into the app and shared by its users.
His write-up puts the consequence in a sentence: "Every mobile app user was given the same hard
coded API Bearer Token, rendering request authorisation useless."

The token was the API's only idea of who was calling. What distinguished one customer's request
from another's was a customer ID on the end of the URL, and the app supplied it. Change the ID and,
in Monie's words, "we are able to access sensitive Personally Identifiable Information (PII) for
that customer": name, date of birth, email address, gender, telephone number, every delivery
address previously used, shareholder number and shares held, referral count, the customer's bar
discount and the identifier behind it. The IDs, he notes, "aren't quite sequential, but certainly
aren't random". The token had been in the app since version 2.5.5 in March 2020, about eighteen
months. Version 2.5.12, released on 13 September, still had it; Monie tested six beta builds before
version 2.5.13 reached the store on 27 September. At publication, Monie said he knew of no
customer or shareholder notification.

Chapter 1's bug was a lookup that never asked whose record it was fetching. This one is a step
earlier: the server never learned who was asking in the first place. Every caller presented the
same credential, so "the caller" was the app, not a person, and any later check of the form "may
this person see this record" had nothing to work with. OWASP calls this **broken authentication**
and puts it second on its API Security Top 10.

## Where the bug lives: a credential that names the app, not the user

Ledger has a mobile app too, and it is how Ben, at Birch, reads Birch's invoices on his phone. Here
is how the app authenticated to `/v1` when v1 was written: the login helper from chapter 1,
`currentUser`, as it stands before this chapter's fix.

```go
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
```

Read the two halves separately. The first half is authentication: it takes the bearer token from
the request and looks it up. The token it finds is `mk_ledger_mobile_…`, the key issued to the
mobile app when v1 shipped, the same one in every copy of the app. The second half is the identity:
because the app's key does not name a person, the middleware reads `X-User` from the request and
trusts it. The app fills that header in. So does anyone else who has the app's key, and everyone
who has the app has the key.

Now recall what chapter 1 built. `LoadInvoiceFor(user, id)` checks that the stored invoice's
tenant is one the user may read. It does its job. It just does it for whichever user the request
claimed to be. Ben's phone sends `X-User: ben` and gets invoice 205. A request that sends
`X-User: alice` with the same app key gets invoice 104. The loader is not broken. It was handed a
user the server never verified.

The fix is that the credential must identify the person, and the server must be the one that says
so:

```go
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
```

Three details matter.

- **The user comes from the credential, not from a header beside it.** A session token is issued
  to one person at login and maps to that person on the server. Nothing the client sends alongside
  it can change who it is. `X-User` is gone.
- **An app key is not a login.** Ledger keeps `mk_ledger_mobile_…` for what it can honestly
  prove: that the caller possesses the shared key. It cannot prove that a real build sent the
  request, much less select a user.
- **Tokens are the server's to revoke.** BrewDog could revoke its shared key at the server, but
  restoring the app's access required a changed client; the disclosure recounts several releases
  before the fix held. A session token lives in Ledger's store with a twelve-hour lifetime and
  can be deleted there.

## If the fix is a login, why is it number two?

Because "the app is logged in" is easy to mistake for "the user is logged in", and several things
keep the mistake alive.

- **The first client was trusted.** v1 was written for one mobile app, built by the same team.
  A shared key was the simplest thing that worked, and the `X-User` header was a convenience for
  that app. Nobody was going to lie to their own server. The design was never revisited as the
  app reached more customers.
- **The credential is invisible from the outside.** Every request carries a valid bearer token, so
  logs, gateways and dashboards show authenticated traffic. Peloton's leak looked like anonymous
  reads; BrewDog's looked like customers using the app.
- **Authorization checks pass.** This is the part that catches teams who did chapter 1's work.
  A per-record check is only as good as the identity it is given, and a test that logs in as
  Alice through the real login flow never exercises a forged identity.
- **The secret is in the binary.** Anything shipped inside an app can be read out of it. A key in
  an app is a key in public; the only question is when someone looks.

## The key you rotated, and the header you forgot

Here is how the fix goes wrong at Ledger.

Ops learns, from a report much like Monie's, that the mobile app's key is on a paste site. They
rotate it: a new `mk_ledger_mobile_…` goes into version 3.1 of the app, the old key is revoked,
and users are told to update. Within a week the old key stops appearing in logs. The report is
closed.

The rotation changed nothing that mattered. Version 3.1 still sends the shared key plus `X-User`,
and the new key is in the new binary. Anyone who cared has it again in an afternoon. Rotation is
the right response to a leaked *secret*; it is no response to a *design* in which the secret was
never the thing that identified the user.

The real fix is the one above: a login flow that issues a session token to a person, and a
middleware that derives the user from that token alone. Ledger ships it in version 3.2 of the app
and in `/v2`. Which leaves the second door.

`/v1` is still serving. It has to, until the old app is gone, and chapter 1 already put its
handlers behind `LoadInvoiceFor`. But `/v1`'s middleware is the old one. It still accepts the app
key, still reads `X-User`, and still hands the loader a user the server never checked. Chapter 1's
two-user test loop passes on `/v1`, because that loop logs in properly and then asks for the wrong
invoice. It never asks for the *right* invoice as the wrong person.

### 1. Find every place a user is decided

Grep for `X-User`, for `currentUser`, for anything that turns a request into a `User`. Ledger
finds two: the new session middleware on `/v2`, and v1's app-key middleware. There is no third,
which is the point of looking: the answer is a list, and the list is short enough to fix.

### 2. Make one middleware, and make v1 use it

v1 and v2 share the session middleware. The old app cannot log in, so `/v1` now answers its
requests with 401 and a message pointing at the update. That is a visible break for users still on
the old app, and it is the correct one: the alternative was continuing to let anyone with the key
be anyone. Ops sets the date, warns the app's remaining users, and moves on.

### 3. Test identity, not only access

Chapter 1's loop asked: can Alice read 205? This chapter adds the question underneath: can a
request *become* Alice without Alice's credential? The table below is this chapter's test, to
run in both modes once its service code exists: the vulnerable build proves the forgery works, the
fixed build proves it does not.

| Route | Credential sent | `X-User` header | Invoice | Vulnerable build | Fixed build |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `GET /v2/invoices/{id}` | Alice's session token | none | 104 | 200 | 200 |
| `GET /v2/invoices/{id}` | Ben's session token | `alice` | 104 | 200, Cedar's invoice | 404, no data |
| `GET /v1/invoices/{id}` | app key | `alice` | 104 | 200, Cedar's invoice | 401 |
| `GET /v1/invoices/{id}` | app key | `ben` | 205 | 200 | 401 |
| `GET /v2/invoices/{id}` | Alice's session token, 13 hours old | none | 104 | 200 | 401 |

The second row is the one to stare at. Ben is properly logged in. He is not attacking the
loader; he is telling the server who he is, and the vulnerable middleware believes him.

### 4. Write down which credentials exist

Ledger now has three kinds: session tokens for people, the shared mobile app key,
and the tenant API keys `ck_cedar_…` and `bk_birch_…` that Cedar's and Birch's own systems use.
Each is recorded with what it proves and what it may select. The tenant keys will matter again
when tenant integrations call Ledger.

<!--mission-->
## Exercise: who does the server think you are?

Use Ledger as it stands after this chapter: `/v2` uses session tokens, `/v1` still serves behind
the same session check, and three kinds of credential exist. You do not need a running service. Here are five middleware
behaviours, written as what the server does with the request before any handler runs:

```text
A. Read bearer token; look up session; user = session.User.
   If none, 401.
B. Read bearer token; if it is the mobile app key, user = X-User header.
   If none, 401.
C. Read bearer token; look up session; user = session.User.
   If X-User is present and differs, user = X-User (support tooling).
D. Read bearer token; if it is a tenant API key (ck_/bk_), user = that
   tenant's service account. Ignore X-User.
E. Read bearer token; if it is a tenant API key, user = the account named
   in X-User, provided that account belongs to the key's tenant.
```

For each one:

1. Say where the identity comes from, and what the server verified about it.
2. Mark it safe or vulnerable.
3. For each vulnerable one, write one test: the credential, the headers, the request, and the
   expected result in the fixed build.

**Check your answer.**

- **A is safe.** The identity comes from the session record on the server. The client supplied a
  token, and the token was issued to one person at login. Test it anyway: Alice's token with
  `X-User: ben` still returns Alice's view.
- **B is vulnerable.** This is Ledger's old v1 middleware, analogous to BrewDog's shared-key
  design; BrewDog's published case selected the customer by URL ID, not an `X-User` header. The
  key proves only its own possession, and the identity comes from a header the client wrote. Test: app key,
  `X-User: alice`, `GET /v1/invoices/104`; fixed build returns 401, not Cedar's invoice.
- **C is vulnerable, and it looks like A.** The first line is identical. The second line lets any
  logged-in user override who they are, because someone needed it for a support tool. Test: Ben's
  token, `X-User: alice`, `GET /v2/invoices/104`; fixed build returns 404, not Cedar's invoice.
  If support staff need to act as a customer, that requires a separate, audited platform-support
  capability, not Dana's tenant-admin role or a header any session can send.
- **D is safe.** The identity is the tenant's service account, decided by the key alone. Birch's
  key can read Birch's invoices and nothing else, because `LoadInvoiceFor` still runs afterwards.
  With Birch's key and `X-User: ben`, `GET /v2/me`, the route that answers "who am I", still
  names the Birch service account.
- **E is vulnerable, and it looks like D.** The difference is one clause: the key selects a tenant,
  and then a header selects a person inside it. The check that the account belongs to the tenant
  is real, and it is not enough: Birch's key with `X-User: ben` is Ben, with no login, no session,
  no expiry. Anyone holding `bk_birch_…` is every Birch user at once. Test with the same key and
  `X-User: ben` against `GET /v2/me`: E names Ben; the fixed D-style rule names the Birch
  service account. An invoice request would not tell them apart, since both get 205 with a 200.

If you marked C safe because it starts the same way as A, you have found the shape of this
chapter's bug: the server did verify a credential. It just let the client finish the sentence.

*Incident from Alan Monie, "Free BrewDog beer with a side order of shareholder PII?", Pen Test
Partners, 8 October 2021. The method follows Colin Domoney, Defending APIs (Packt, 2024).*
