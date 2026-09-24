# Left On

<!-- claims to gate in checks/claims/08.tsv: UpGuard, "By Design: How Default Permissions on Microsoft Power Apps Exposed Millions", published 23 Aug 2021, https://www.upguard.com/breaches/power-apps. Facts used: Power Apps portals expose an OData API at /_odata; a list's data was anonymously readable when the OData feed was enabled and the list's "Enable Table Permissions" setting was not set; that setting was off by default on every list ("all lists having table permissions disabled by default"); table permissions on the table itself do block anonymous access but lists ignore them until the list setting is on; Microsoft documentation quote "To secure a list, you must configure Table Permissions ... and also set the Enable Table Permissions Boolean value on the list record to true"; discovery 24 May 2021; found portals by subdomain enumeration of powerappsportals.com / powerappsportals.us / microsoftcrmportals.com and search engines; visiting /_odata listed the lists, visiting a list either showed data or a forbidden message; report to MSRC 24 Jun 2021; case closed 29 Jun 2021 as "determined that this behavior is considered to be by design"; over a thousand anonymously accessible lists across a few hundred portals; 47 entities notified; 38 million records in total; notifications from 2 Jul 2021; Microsoft's own Global Payroll Services portal, 332,000 records (names, @microsoft.com addresses, phone numbers, employee IDs), no longer public by 16 Jul after an abuse report on 15 Jul; all but one of the remaining Microsoft portals closed by 19 Jul; Microsoft released the Portal Checker and made table permissions enabled by default on newly created portals; UpGuard agrees with Microsoft that it is "not strictly a software vulnerability". Rejected pointers: All in One SEO CVE-2021-25036 (Jetpack, Marc Montpas, Dec 2021: case-insensitive REST route bypass of validateAccess — an authorization bug, not a setting); Domoney's home-router case (command injection on an unauthenticated /cgi-bin/adm.cgi endpoint — broken authentication plus injection, Domoney's own root-cause line). -->

In May 2021 an analyst at UpGuard found that a Microsoft Power Apps portal was answering
requests for its data without asking who was making them. Power Apps portals are low-code
websites that companies and public bodies build on Microsoft's platform: a vaccination sign-up
form, a job application page, a dealer self-service site. Behind the pages, each portal can
expose an OData API, and every list it publishes is reachable at a URL of the form
`example.powerappsportals.com/_odata/mylist`. Visit `/_odata` and the portal names its lists.
Visit a list and you get either the rows or a message saying access is forbidden.

Which of those two you got depended on one setting. Microsoft's documentation said it plainly:
to secure a list, you configure table permissions for the table *and* set the list's
"Enable Table Permissions" value to true. The table permissions did their job. But a list
ignored them, including custom table permissions, until that one Boolean on the list was turned on, and
it was off by default on every list. Nobody had removed a check. The check was there, wired
correctly, and switched off.

UpGuard enumerated portal subdomains, walked the `_odata` endpoints, and found over a thousand
anonymously readable lists across a few hundred portals. On 24 June they reported it to the
Microsoft Security Response Center with URLs for portals exposing sensitive data, among them
COVID-19 contact-tracing and vaccination portals run by American government bodies and a
portal with job applicants' Social Security numbers. On 29 June Microsoft closed the case: the
analyst had "determined that this behavior is considered to be by design." So UpGuard spent July
notifying the owners one by one: 47 entities, 38 million records. One of the owners was
Microsoft. Its Global Payroll Services portal had a list of 332,000 records with names,
`@microsoft.com` addresses, phone numbers and employee IDs; it went private the day after UpGuard
filed an abuse report. By the time the write-up was published, on
23 August, Microsoft had released a Portal Checker that flags lists open to anonymous access and
had announced a safer default for newly created portals, without a rollout date in this account.

UpGuard's title for the piece was *By Design*, and they agree with Microsoft that what they found
was not, strictly, a software vulnerability. That is the point. Every check the platform offered
worked. What failed was a value whose safe state was available and whose default was the other
one. OWASP calls this **security misconfiguration**, API8, and the category is wider than
defaults: debug modes left on, error bodies carrying stack traces, CORS that reflects any origin,
HTTP methods nobody meant to serve, optional TLS. The code is fine; the configuration is not.

## The check ran. The value was wrong

Ledger's version of the Power Apps list is a declaration. Since the chapter on function-level
authorization, every route carries an `Access` level, `Public`, `User` or `TenantAdmin`, and a
route without one refuses to register. The middleware enforces the level on every request. That
is the machinery UpGuard found working on Power Apps: enforcement is not the problem.

Ops has a liveness probe, `GET /v2/health`, which the gateway polls and which returns
`{"ok":true}`. It is `Public`, correctly: a load balancer has no session, and the body says
nothing. Later someone needed a richer probe for the deploy dashboard and copied the entry:

{{excerpt:ch08-vulnerable-health-decl}}

`GET /v2/admin/health` returns the running version, the commit, the hosts the build is
configured to serve (`api.ledger.example` and `ledger-staging.internal`), and the value of the
`Debug` flag. The route sits under `/v2/admin`, so it reads as admin-only. It is not. The
declaration next to it says `Public`, the middleware honours the declaration, and the test that
walks the route table checks that every route *has* a level, which this one does. The researcher
who reads the response learns that Ledger has a staging host, what it is called, and that the
production build is running with `Debug` on.

The fix is a value, not a check:

{{excerpt:ch08-fixed-health-decl}}

Two details decide whether the fix is real.

- **The declaration moved, and the body shrank.** Dana can still see that the service is up and
  which version it is. Hosts and flags are not a tenant's business at any level; they leave the
  response entirely rather than sit behind a role a later copy-paste might loosen again.
- **`/v2/health` stays `Public`.** The liveness probe is this route's pair: same declaration,
  opposite verdict, because its body carries nothing. A route being `Public` is not the bug. A
  `Public` route with something to say is.

## Why "just flip it" is not the fix

The Power Apps owners each flipped one Boolean and the data went private. Microsoft's fix was
different: it changed the default and shipped a checker. That distinction is the whole chapter.

- **Nobody tests a setting.** Ledger's chapter 5 test asserts that every route declares a level.
  It cannot tell a correct `Public` from a wrong one, because the table is the only statement of
  intent there is. UpGuard reports that several government bodies had run security reviews of
  their portals without finding this, presumably, they say, because it had never been publicised
  as a concern. A setting nobody has named is a setting nobody checks.
- **Defaults are copied, not chosen.** The admin probe was `Public` because the liveness probe
  was the nearest working example. Every Power Apps list was open because that is what a new
  list looked like. A default is a decision made thousands of times by people who did not know
  they were making it.
- **A setting can undo a check without touching it.** That is the second door, below.
- **The exposure is quiet.** A public route that answers 200 raises no alarm, because nothing
  was denied. UpGuard found the lists by visiting URLs, as a load balancer would.

## The setting that reopens a door you already closed

The second path runs through a repair from the first chapter. Chapter 1 made `LoadInvoiceFor`
return "not found" for an invoice the caller may not see, and the handler answers 404 for a
missing invoice and a forbidden one alike, so the route cannot be used to learn which invoice
numbers exist. That decision is still in force. It is also, in one build of Ledger, undone by a
setting that never touches the loader.

`Debug` was added for staging. When it is on, the error writer asks the store for an internal
denial reason and appends it to the error body, so a developer on `ledger-staging.internal` can
see why an invoice request failed:

{{excerpt:ch08-debug-error-writer}}

The loader still runs. The decision is still 404. But the body now says
`{"error":"not found","reason":"invoice 205 belongs to tenant birch"}`, and Alice, who asked for
205 from Cedar, has just been told that it exists and whose it is. `Debug` was on in production
because staging's configuration was copied when the two were split, and nobody tested
production's values because nobody thought of them as code. The admin probe had been advertising it.

That is UpGuard's shape. Table permissions on a Power Apps table are the loader: real, enforced,
correct. The list's Boolean is `Debug`: a value elsewhere that decides what reaches the caller.

## Configuration is code, so it gets a test

Ledger's repair has two parts, and neither is "turn it off."

### 1. Write the production settings down, in the language the service is written in

{{excerpt:ch08-production-settings}}

`Settings` is a struct, constructed in one place, with `Debug false`, the CORS origin list
holding only Ledger's own web application, and `PublicRoutes` naming the three routes that may
serve callers without a session: the liveness probe and the two one-time-code routes from the
chapter on resource consumption, `POST /v2/auth/otp/request` and `POST /v2/auth/otp/verify`. A
staging build constructs its own `Settings` with `Debug true` and the staging host. Production
cannot inherit staging's values, because there is no inheritance: there are two literals, and the
differences between them are a diff.

### 2. Assert every value, and walk the route table against the allow-list

{{excerpt:ch08-settings-test}}

This is where chapter 5's repair does new work. That chapter's test walks the route table and
fails on a route with no `Access`. This one walks the same table and fails on a route whose
`Access` is `Public` and whose method and pattern are not in `Settings.PublicRoutes`. The
copy-pasted admin probe fails it at once: `GET /v2/admin/health` is `Public` in the table and
absent from the list. The same test asserts `Debug` is false, that no CORS origin is `*`, and
that a production error body is exactly `{"error":"not found"}`, by sending Alice's request for
205 and reading the bytes.

It is Microsoft's Portal Checker with one difference: the checker runs on demand against a
portal someone remembers to check; this runs on every build, against the values that build will
ship with, and fails the build instead of producing a report.

### 3. Make the safe value the default

An unset `Debug` is off, because the zero value of a `bool` is the safe one. `Access` has no
zero value, chapter 5's rule, and `Public` now needs justifying in a second place. Microsoft's
fix was the same idea one level up: new portals start with table permissions on, and an owner
who wants an anonymous list has to say so. The setting still exists. The default no longer
chooses for you.

<!--mission-->
## Exercise: which value is the exposure?

Ledger as described: Alice is an ordinary Cedar user, Dana is Cedar's admin, invoice 205
belongs to Birch. Below are five configurations, each a route declaration or a `Settings` value
paired with one request. For each: say what the enforcement does, then what the caller
receives, and mark it exposure or not.

```text
A.  GET /v2/health         Access: Public        caller: none
    body: {"ok":true}

B.  GET /v2/admin/health   Access: Public        caller: none
    body: {"version":"…","hosts":["api.ledger.example","ledger-staging.internal"],"debug":true}

B'. GET /v2/admin/health   Access: TenantAdmin   caller: Dana
    body: {"version":"…"}

C.  Settings{Debug: true}  GET /v2/invoices/205  caller: Alice

C'. Settings{Debug: false} GET /v2/invoices/205  caller: Alice
```

This is this chapter's test, to run in both modes once the service code exists.

| Case | Route or setting | Caller | Vulnerable build | Fixed build |
| :--- | :--- | :--- | :--- | :--- |
| A | `GET /v2/health`, `Public` | none | 200, `{"ok":true}` | 200, `{"ok":true}` |
| B | `GET /v2/admin/health`, `Public` | none | 200, version, hosts, `debug:true` | 401, no body detail (route is `TenantAdmin`) |
| B' | `GET /v2/admin/health`, `TenantAdmin` | Dana | 200, version, hosts, `debug:true` (vulnerable build has no such declaration; Dana sees the public body) | 200, `{"version":"…"}` only |
| C | `Debug: true`, `GET /v2/invoices/205` | Alice | 404, body includes `reason: invoice 205 belongs to tenant birch` | 404, `{"error":"not found"}`; a separate settings assertion rejects `Debug: true` for production |
| C' | `Debug: false`, `GET /v2/invoices/205` | Alice | 404, `{"error":"not found"}` | 404, `{"error":"not found"}` |

**Check your answer.**

- **A is not an exposure, and it is the decoy.** A `Public` route is the first thing to suspect
  after this chapter, and this one is `Public` on purpose. Trace it: no session, the middleware
  admits the request because the declaration says so, the handler writes `{"ok":true}`. The
  caller learns that the service is up, which serving it at all already says. It is in
  `PublicRoutes`, so the settings test passes. The question is never whether a route is public;
  it is what a public route says.
- **B is the exposure, and A is its pair.** Same declaration, same absence of a session, same
  path through the middleware. The difference is entirely in the body: version, two hostnames
  including an internal one, and the fact that `Debug` is on. In the fixed build the route is
  `TenantAdmin`, so a request with no session stops at `currentUser` with 401; and the settings
  test would have failed the build before that, because a `Public` route not in `PublicRoutes`
  is a failure.
- **B' is B with the right caller and the right declaration.** Dana's session resolves; the
  middleware finds `TenantAdmin` and she clears it; the handler answers with the version and
  nothing else. Hosts and flags are gone from the response, not hidden behind her role.
- **C is the second exposure, and it never touches the route table.** Alice's request reaches
  `LoadInvoiceFor`, which finds 205, compares its tenant to hers, and returns not-found. The
  handler answers 404, exactly as chapter 1 intended. Then the error writer reads `Debug`, finds
  it true, asks the store for a denial reason, and appends it. The check ran; the setting leaked
  its verdict. In the fixed build the same request returns an opaque 404, while a separate
  settings assertion rejects `Debug: true` in production.
- **C' is C with one value changed.** Same request, same loader, same 404, and a body of
  `{"error":"not found"}`. Alice cannot tell 205 from an invoice number that was never issued.
  One Boolean separates C from C', and no handler, middleware or loader differs between them.

If you marked A an exposure, you have found every `Public` route, which is the grep, not the
judgement. The judgement is the allow-list.

*Incident from UpGuard, "By Design: How Default Permissions on Microsoft Power Apps Exposed
Millions", 23 August 2021. The method follows Colin Domoney, Defending APIs (Packt, 2024).*
