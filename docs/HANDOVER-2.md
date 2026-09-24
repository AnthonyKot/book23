# Lane A batch 2 implementation handover

The reviewed Chapters 00–10 provide the prose and source gates. This handover tracks the
remaining cumulative Ledger implementation, one chapter per run.

| Chapter | State | Next evidence |
|---|---|---|
| 01 | Complete for batch 2; refund/list cases and route coverage tested | Author reconciles schematic exercise wording |
| 02 | Complete for batch 2; session identity and exercise tested | Author reviews identity-probe wording |
| 03 | Complete for batch 2; views, patches, and guard tested | Author reconciles mixed PATCH wording |
| 04 | Complete for batch 2; challenge, lookup, and page limits tested | Author aligns OTP excerpt wording |
| 05 | Complete for batch 2; access declarations, admin routes and exercise tested | Author aligns vulnerable-build declaration wording |
| 06 | Complete for batch 2; tenant allowance, approval and exercise tested | Author reviews thin confirm wrapper and reminder simulation |
| 07 | Complete for batch 2; shared egress, webhook/PDF paths and exercise tested | Author checks logo-fixture wording |
| 08 | Complete for batch 2; settings, probes, debug contrast and exercise tested | Author aligns public-route count and login |
| 09 | Pilot exists; integration through 02–08 pending | Cumulative app and regression tests |
| 10 | Reviewed prose and brief; service pending | Partner response and refund tests |

Chapter 01's exercise is schematic: it omits the refund quote amount, and its answer both
says the injected confirm ID is rejected and says confirm ignores it. The executable
contract for this pass is in `briefs/LANE-A-BATCH-2.md`: quote amount is pence, using the
Chapter 06 £400 example; fixed confirm rejects an extra `invoice_id` and a valid confirm
uses the stored quote target. The author should reconcile that wording before treating the
Chapter 01 exercise as a literal wire example.

## Chapter 01 — completed in batch 2

- `NewApp` now registers `GET /v2/invoices?tenant=X` and the refund quote/confirm pair.
  Both modes authorize the quote through `LoadInvoiceFor`. Vulnerable confirm uses a fresh
  client-supplied `invoice_id`; fixed confirm rejects that field and acts on the quote's
  stored invoice. A valid confirm replays idempotently. The first quote on invoice 104 is
  `q-771` even if another invoice was quoted first; later IDs are internal and unnamed.
- `service/ch01_test.go` asserts exercise A–E in both modes. The injected confirm changes
  Birch's refunded amount only in vulnerable mode; the fixed response is 400 and neither
  invoice changes. The near-identical list routes distinguish server-derived from
  caller-selected tenant filters. `TestChapter01RegisteredInvoiceRoutesCovered` compares
  every registered Chapter 01 GET route with Alice and Ben read cases, so a new registered
  read route without both callers fails. POST quote/confirm are covered as a sequence.
- `verify.sh` requires the new test groups, validates all existing 00–10 claim ledgers
  against manifest-backed archives, and rejects placeholders in built Chapters 01 and 09.
  Chapter 01 has no `briefs/01.md` to retire; its original printed excerpts remain intact.
- Fixture limits: the in-memory store has no durable payment ledger or concurrent refund
  transaction yet. Chapter 06 must make its allowance and refund record atomic. The current
  Chapter 09 pilot still lacks Chapters 02–08 and intentionally retains its original route
  inventory until the cumulative integration pass. No new canonical value was added.
- Validation: `cd service && go test ./...`, `./verify.sh` and `git diff --check` pass.
  The verifier still reports 31 pending code blocks in the unbuilt chapters; Chapter 01
  has no placeholder left.

## Chapter 02 — completed in batch 2

- `NewChapter2App` keeps the Chapter 01 loader and list repairs in both modes. One identity
  middleware supplies the caller to the existing handlers across `/v1` and `/v2`; the Chapter 01
  marked Go blocks and the Chapter 09 pilot remain unchanged. Vulnerable mode trusts `X-User`
  alongside a session token (even after expiry), shared mobile key, or same-tenant integration key. Fixed mode derives
  a person only from a live server-side session, ignores `X-User`, and treats tenant keys as
  service accounts. The shared app key alone cannot select any identity.
- `POST /v2/auth/login` issues a session from fixture credentials. The store records issue time,
  expires fixed-mode sessions at 12 hours, and can revoke a token. Tests inject a fake clock,
  including the exact boundary and a 13-hour-old token through both API versions. The fixture
  passwords and demo key suffixes are test-only strings, not new canonical book values.
- `GET /v2/me` is the person-scoped identity probe needed by the reviewed D/E exercise: Birch's
  key plus `X-User: ben` names Ben in vulnerable mode and the Birch service account in fixed mode.
  Both cases would otherwise read Birch invoice 205 with status 200, hiding the distinction.
  `service/ch02_test.go` asserts all exercise A–E outcomes in both modes, old v1 header behavior,
  registered-route rejection of the app key, and survival of Chapter 01's invoice, list, and
  refund repairs.
- Exact source excerpts `ch02-vulnerable` and `ch02-fixed` replace both Chapter 02 placeholders;
  the incident header now points at `checks/claims/02.tsv`. `verify.sh` requires all Chapter 02
  test groups, rejects placeholders, and checks the printed excerpts against their marked source.
  The existing claim-ledger/archive validation continues to cover Chapters 00–10.
- Author note: the chapter calls `currentUser` a Chapter 01 "login middleware", although that
  stage uses an opaque-token helper and has no login route. The new route is Chapter 02's fixture
  login. The exercise describes a person-scoped identity probe without naming its concrete route;
  the implementation uses `/v2/me`. These are prose alignment choices for the author.

## Chapter 03 — completed in batch 2

- `NewChapter3App` composes the Chapter 01 loader/refund repairs and Chapter 02 session identity
  repair in both modes. The vulnerable Chapter 03 routes encode a copy of the full invoice row,
  accept matching stored fields on invoice PATCH, and let profile PATCH set `is_admin`.
  Fixed routes encode `ViewFor(user, inv)` after `LoadInvoiceFor`, use strict
  `InvoicePatch{Reference}` and `ProfilePatch{DisplayName}`, and bind the PDF to an
  `InvoiceView` that has no contact or internal fields. Dana's admin view has contact fields;
  neither view has `CollectionsNote` or `Margin`.
- `Invoice.MarshalJSON` rejects raw row encoding. The vulnerable read uses a named `rawInvoice`
  alias to demonstrate the bypass explicitly; historical Chapter 01/02 and the partial Chapter 09
  pilot use a separate four-field legacy response so their previously printed handlers and
  behavior stay intact. A direct raw marshal and an accidental raw `writeJSON` call fail in tests.
- The chapter adds only fixture details for the existing invoices: distinct invoice numbers,
  line-item labels, dates, customer contact examples, references, notes and margins. The canonical amounts remain
  180000 and 64000 pence. These extra strings and internal margins are illustrative, not new
  cross-chapter canonical values.
- `service/ch03_test.go` asserts the chapter's read, list, PDF, invoice PATCH and profile PATCH
  contrasts in both modes. It also checks direct raw encoding failure, every registered invoice
  read route, cross-tenant reads and PATCH, forged identity rejection, session expiry, and the
  earlier refund repair. Five exact excerpts (`ch03-vulnerable-read`, `ch03-view`,
  `ch03-vulnerable-patch`, `ch03-patch`, `ch03-marshal-guard`) replace the Chapter 03 placeholders.
  `verify.sh` requires these test groups, validates their source equality, and continues checking
  all archived incident claims.
- Author note: the exercise's D row says a mixed `{"reference":"PO-9","status":"paid"}` patch
  updates the reference in fixed mode. The required `DisallowUnknownFields` policy rejects that
  entire request with 400, leaving both fields unchanged. A reference-only patch succeeds. The
  prose should reflect that distinction. The vulnerable read's `rawInvoice` alias also deserves
  one short explanation beside the excerpt: the raw `Invoice` type itself now refuses JSON.
  The chapter still calls the implementation proposed and its tests future work; the author
  should update that framing. The rendered Chapter 03 page remains stale until the batch's
  deferred site build after Chapter 10; the Markdown source already has all five excerpts.

## Chapter 04 — completed in batch 2

- `NewChapter4App` keeps Chapters 01–03's repairs in both modes. OTP request issues a test-only
  six-digit code challenge with a ten-minute window. Vulnerable verification has only an
  eight-per-minute IP guard; fixed verification also counts wrong guesses against the active
  `(account_id, challenge_id)`, locks on the fifth failure, rejects reissue while locked, and
  spends a challenge on success. An unlocked fixed reissue returns the same challenge without
  resetting attempts. A successful code issues Chapter 02's twelve-hour session.
- `GET /v2/invoices?email=…` and `GET /v1/invoices?email=…` both use the Chapter 01 tenant scope
  and Chapter 03 views. Vulnerable v2 has only a 30/min IP budget at the gateway; vulnerable v1
  has no lookup budget. Fixed mode registers both routes with separate 30/min tenant-and-route
  budgets at the gateway and in their shared handler. Invoice-by-ID is a separate route and does
  not consume this budget. `?limit=N` caps at 50 only in fixed mode; neither mode treats the page
  cap as a frequency budget.
- `service/ch04_test.go` exercises all OTP A/A'/A'' and lookup B/C sequences in both modes,
  including same-IP and rotating-IP probes, exact ten-minute reissue, challenge use, a direct
  IP-guard probe, and direct proofs of the gateway and handler lookup budgets. It injects 59
  extra Cedar rows to make a 60-row page solely within the page test, then sends 10,000
  50-row requests to prove the size cap does not limit frequency. Regression checks cover the
  earlier loader, session, view, typed PATCH, and refund repairs.
- The fixed values from `CONTEXT.md` remain unchanged. The eight-per-minute IP guard,
  `731842` OTP code, sequential challenge IDs, and extra page rows are fixture-only values.
  No real text is sent and the in-memory limiter is not a production rate limiter.
- The three printed excerpts (`ch04-otp-vulnerable`, `ch04-limiter-per-ip`,
  `ch04-otp-fixed`) exactly match marked Go blocks, and `verify.sh` now requires their test
  groups and checks their equality while continuing to validate all claims archives.
- Author note: the brief's older handler list says vulnerable v1 email lookup is unregistered,
  but its exercise C and the reviewed chapter require v1 to serve without a budget. The service
  follows the exercise. The two OTP excerpts are challenge-store verification methods called by
  the HTTP handler, so the surrounding prose's word "handler" should be made precise. The
  exercise's stated snapshot already has the OTP repair while the vulnerable build in its
  executable table still needs the OTP flaw; the two modes represent the chapter-wide before
  and after states, and the author should make that distinction clear. The
  Chapter 04 Markdown is filled; rendered HTML awaits the deferred batch site build.

## Chapter 05 — completed in batch 2

- `NewChapter5App` composes Chapters 01–04 in both modes. The Chapter 05 route inventory contains
  eighteen routes, each with explicit `Public`, `UserAccess`, or `TenantAdmin` metadata in
  `service/routes.go`. Go already has a `User` model type, so the Go constant for the prose's
  `User` level is named `UserAccess`. Construction panics if a registered route lacks a
  declaration or a declaration has no route.
- The shared Chapter 05 authorization layer matches the registered method and path, derives the
  caller from Chapter 02's session, checks invoice or target-user tenant scope first, then checks
  the route's access level. Missing or cross-tenant targets answer 404; same-tenant wrong-role
  requests answer 403 in fixed mode. Vulnerable mode deliberately skips only the
  `TenantAdmin` role decision. The route table remains populated in both modes so route
  composition is identical; the chapter's assertion that vulnerable routes have “none” should
  be described as “not enforced” if the prose is later aligned to the executable fixture.
- The new DELETE, admin invoice void, admin user list, and admin user-edit routes expose the
  Chapter 05 contrast. User edits update the session store's user record and all live sessions,
  so vulnerable Alice's promotion changes her subsequent Chapter 03 view; typed self-edit still
  rejects `role`. Login and OTP issuance read the same user record after an edit. The fixture
  keeps in-memory state and introduces no new canonical numbers.
- `service/ch05_test.go` asserts every exercise A–F result in both modes, including row mutation,
  admin view exposure, and the 404-before-403 order. It also exercises the void route, missing
  declarations, all route access levels, and earlier loader, session, PDF view, OTP, page and
  profile repairs. The three printed excerpts match their marked Go blocks; the fourth requested
  marker, `ch05-declares-test`, is in the test file but has no placeholder in the drafted chapter.
  `verify.sh` requires the test groups, excerpt equality and no Chapter 05 placeholders.
- Author note: the printed vulnerable DELETE includes a redundant `LoadInvoiceFor` check because
  the common route layer has already scoped the invoice before reaching the handler. This keeps
  the chapter's local counterexample legible, while the fixed handler consumes the scoped row.
  The chapter currently says the vulnerable build has no declarations; in the service, the
  declarations exist but the privileged role check is skipped. The observable failure and exercise
  outcomes are the same. Rendered Chapter 05 HTML awaits the deferred build after Chapter 10.

## Chapter 06 — completed in batch 2

- `NewChapter6App` retains Chapters 01–05's fixed behavior in both modes. The Chapter 06
  difference is at refund confirm: vulnerable mode accepts three fresh £400 quotes and records
  £1,200 for Cedar; fixed mode records two and parks the third at £800 with HTTP 202 and
  `{"status":"refund_needs_approval"}`. The first quote on invoice 104 remains `q-771`,
  and a second confirm of that quote returns the original refund without recording another.
- `Store.ConfirmRefund` locks the quote, invoice, refund list and tenant allowance decision
  together. It sums recorded refunds for the quote's stored tenant on the UTC calendar day.
  Cedar and Birch each have the established 100000-pence fixture allowance. A parked quote stays
  parked after midnight; Dana's `POST /v2/refunds/{quote}/approve` records it only after the
  Chapter 05 tenant-scope check and `TenantAdmin` role check. Alice sees 403 for a Cedar quote;
  Ben sees 404. The rejected per-user counter is compiled but not registered as a mode; its
  integration-identity gap is asserted directly in the test.
- The exercise's 50-quote decoy leaves all money unmoved. The reminder route is implemented as
  a deterministic fixture: vulnerable mode records 50 reminder events, fixed mode records the
  first and returns `reminder_needs_review` for the rest. It sends no email. Quote velocity is
  counted per tenant per UTC hour and exposed to the test through a store method; there is no
  Ops HTTP route. UTC day/hour boundaries are fixture policy because the book does not specify
  a tenant timezone. No new canonical number was introduced.
- `service/ch06_test.go` covers the A–E exercise contrasts, the Ben cross-tenant row, the
  integration-key and Dana sequences, approval scope and role, 50 open quotes, 50 reminders,
  next-day behavior, concurrent confirms, and earlier access, identity and loader repairs.
  `go test ./...`, `go test -race ./...`, `./verify.sh`, and `git diff --check` pass. Four printed
  excerpts match their marked Go source; `verify.sh` requires them and rejects remaining
  Chapter 06 placeholders. Rendered HTML awaits the batch build after Chapter 10.
- Author note: the printed vulnerable confirm block contains its unrestricted store method,
  where the per-request checks live. The fixed confirm wrapper delegates to the later printed
  `Store.ConfirmRefund` method. Commit `69c393b` aligned the prose with that call path and
  removed the stale future-tense sentence. Reminder “emails” are simulated events in this
  fixture, as expected for an offline test.

## Chapter 07 — completed in batch 2

- `NewChapter7App` retains Chapters 01–06's fixed behavior in both modes. The Chapter 07
  vulnerable mode uses the injected default-behavior HTTP client for the webhook test and
  tenant-logo fetch during `/v1/invoices/{id}/pdf`; fixed mode gives both routes the same
  `egress` instance. The PDF handler still calls `LoadInvoiceFor` before any logo fetch and
  renders the Chapter 03 public view. A missing or blocked logo leaves the invoice export
  available without contact fields.
- `egress.Fetch` requires HTTPS, resolves the hostname once, rejects the whole DNS answer set
  if any address is private, loopback, link-local or reserved (including RFC 5737 test ranges),
  and dials the approved IP with the original URL hostname retained for TLS verification.
  It disables environment proxies and returns redirects without following them. The production
  constructor uses the Go resolver and dialer; tests inject both. The fixture's
  `https://logo.cedar.example/mark.png` is a new illustrative Cedar branding value, not a
  canonical number or a live endpoint. The static deny ranges are a conservative snapshot of
  IANA's [IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry) and
  [IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry) special-purpose registries;
  an actual deployment would need maintained address policy and network enforcement as well.
- `service/ch07_test.go` covers the five exercise cases in both modes. The HTTP response
  fixture shows the redirect reaching a synthetic metadata response only in vulnerable mode;
  the link-local case proves a vulnerable dial attempt and a fixed pre-dial refusal, without
  claiming a TLS handshake or token leak. A separate direct dial-callback test proves pinning;
  a canned `RoundTripper` alone cannot prove it. Mixed DNS answers, reserved address classes,
  the PDF logo redirect and link-local second path, and earlier loader, identity, view and role
  repairs are also tested. No test opens a real socket.
- Four printed excerpts match marked Go blocks. `verify.sh` requires the Chapter 07 test
  groups, rejects placeholders in the built Markdown, and checks excerpt equality. Site HTML
  remains deferred until the batch build after Chapter 10.
- Fixture limit: the PDF renderer models the logo fetch but does not embed an image in its
  minimal text-only PDF response. Webhook delivery itself is not implemented, although the
  chapter mentions a delivery path that forwards a response body; the author should mark that
  as outside the executable fixture or add it in a later build. The tested outbound paths are
  the webhook test and PDF logo fetch. These are teaching fixtures, not claims about a deployed
  network boundary.

## Chapter 08 — completed in batch 2

- `NewChapter8App` composes Chapters 01–07's fixed behavior in both modes. Vulnerable
  settings enable `Debug` and declare the admin health probe `Public`; fixed settings
  disable `Debug`, declare the probe `TenantAdmin`, and return only a version field.
  Both modes retain the harmless public liveness probe. The error writer runs after
  the invoice scope check, so a cross-tenant request remains 404 while only the
  vulnerable body exposes the stored tenant. Fixed mode writes the exact opaque JSON
  body `{"error":"not found"}`. A missing invoice has no tenant-specific reason.
- `validateProductionSettings` rejects `Debug:true`, a wildcard CORS origin, a
  public route absent from `PublicRoutes`, or an allow-listed route missing from
  the public route table. The fixed app validates its settings when constructed.
  The production web origin `https://app.ledger.example` is an illustrative new
  fixture value; no canonical number changed. The version and commit values are
  deliberately non-numeric fixture strings.
- `service/ch08_test.go` asserts exercise A, B, B', C and C' in both modes, including
  an extra vulnerable instantiation with `Debug:false`; it checks the validator's
  rejection of the vulnerable public admin route and earlier identity, view,
  loader and role repairs. Five printed excerpts match marked Go source, and
  `verify.sh` requires their tests and excerpt equality. HTML is deferred to the
  batch site build after Chapter 10.
- **Author alignment needed:** Chapter 8 prose says production has exactly three
  public routes. The cumulative service also keeps Chapter 2's public password
  login (`POST /v2/auth/login`) so the existing session exercise remains usable.
  Production's allow-list therefore has four entries. Either acknowledge the
  login fixture in the prose or explicitly retire that route in a later design;
  do not silently omit it from the validator. The code excerpt makes this
  discrepancy visible. The setting contrast and all five exercise outcomes hold.

Next: Chapter 10 partner response boundary, then Chapter 09 cumulative integration through
Chapters 02–08, as requested on 2026-09-24. Do not push this implementation branch from Lane A.
