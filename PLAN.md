# Book 23 — plan (2026-09-23, draft for the author's decision)

Working title (author, 2026-09-23): **Security Rebook**. Earlier candidates *Who's Asking* / *Every Door to the Table* stay as chapter-title material.

## 1. What this book is

A book of API break-ins, one per chapter, grown from the Book 11 essay the author liked most
("The Lookup That Never Asks Who's Asking", `~/book11/essays/defending-apis.md`). The source
backbone is Colin Domoney, *Defending APIs* (Packt, 2024), already extracted in
`~/book11/workspace/defending-apis/` (95k words, page boundaries kept; corpus record in
`~/book11/corpus/defending-apis/`). Its spine is the OWASP API Security Top 10 (2023), which
Domoney's chapters 3, 7 and 9 follow.

**Intended reader:** a developer or junior AppSec practitioner who can follow a Go handler and an
HTTP request, but has not yet learned to trace authorization and trust decisions across an API.
After a chapter, that reader should be able to name the untrusted input, the missing check, the
second route or step that escapes a local patch, and a test that distinguishes the two. This is
not a Go introduction or a claim that the example service is production-ready.

The essay worked for five reasons, and each becomes a rule:

1. **A real incident with a checkable detail.** Coinbase, one edited source account in a trade
   request, $250,000. Use a number only when the primary record supports it. Not a taxonomy.
2. **The bug fits in fifteen lines**, shown vulnerable then fixed, with the three details that
   decide whether the fix is real.
3. **"If the fix is one line, why is it number one?"** The structural reasons the one line goes
   missing (auth is global, authz is per request; step two trusts step one; tests use one user).
4. **A worked example with a twist**: the route you fixed and the one you forgot (v1, the PDF
   export, the list filter). The reader learns the *question* ("which routes reach this data?"),
   not the patch.
5. **An exercise with a plausible decoy and a contrasting pair**, doable without a running
   system, with a worked answer that explains the different outcomes. The essay's five-handler
   count and closing trap sentence are examples, not requirements.

What is different from the siblings: Book 17 (*Ten Ways In*) is the web OWASP Top 10 told through
big public incidents with a folk-vs-record ledger; Book 21 is the AppSec career. Book 23 is
**code-level, API-only, and executable**: every chapter's handlers exist as real code and every
two-user test in the book actually runs. Book 17's incident register (Capital One, MOVEit, Equifax,
xz, etc.) is off limits here; no overlap.

## 2. The spine: one service, attacked chapter by chapter

The Book 2 lesson from today applies: bridges don't connect chapters, a running system does.

**The service** is the essay's billing API, grown just enough to have every door the Top 10 needs:

- Tenants: **Cedar** and **Birch**. Users: **Alice** (Cedar), **Ben** (Birch), one Cedar admin.
- Resources: invoices (104 Cedar, 205 Birch), refunds (quote → confirm), PDF export, a customer
  profile, an API key per tenant, a webhook URL per tenant, a public price-list endpoint.
- Surfaces: `/v2` (current), `/v1` (an older route still in use), a mobile client, a partner/third-party feed the
  service *consumes* (exchange rates), an API gateway in front.
- Implementation: **Go** (author's decision 2026-09-23), standard library `net/http` plus
  `net/http/httptest` for the two-user tests; no framework, so every check is visible in the
  handler. Chapter 1 carries over the Book 11 essay's request → missing check → shared loader
  method and code shape in real Go handlers; its real incident is selected separately.

Rules of the spine (as in `~/book2-spine/docs/SPINE-BRIEF.md`): fixed names and numbers set once and
reused; later chapters may say "the loader from chapter 1" and mean a specific function. The
fixed service's state accumulates: chapter 1 protects `/v1` with the shared loader but keeps it
serving an old client; chapter 9 discovers the unlisted staging host, inventories every route,
and finally retires `/v1`. The gateway inventory from chapter 9 is what chapter 11 monitors.
Correctness beats the spine: where a class of bug doesn't fit billing, the chapter says so and
uses the smallest honest extension.

## 3. Chapter register (OWASP API Top 10:2023 order; candidate incidents need a primary-source and category-fit check before drafting)

| # | Class | Real incident (candidate) | The one-line bug on the service | The twist (door you forgot) | Choke point built |
|---|---|---|---|---|---|
| 1 | API1 BOLA | Peloton 2021 — selected after [comparing Peloton, USPS and Tinder](docs/CH01-INCIDENT-CHOICE.md); the researcher documents editable IDs in a request that exposed other members' information after login was required | `store.Invoice(id)` never asks whose | `/v1/invoices/{id}/pdf`; refund confirm trusts body | `LoadInvoiceFor(user, id)` |
| 2 | API2 Broken authentication | BrewDog app, hard-coded bearer token (Pen Test Partners 2021; Domoney case 3) | one shared token in the mobile binary; reset flow that returns the token | rotating the token ≠ fixing the design; the reset endpoint is the second door | per-user tokens, reset that never echoes the secret |
| 3 | API3 Object property level (excessive exposure + mass assignment) | shipping-company API returning full recipient details (Domoney case 1) — primary source to find | handler encodes the whole struct; PATCH decodes into it, `IsAdmin` included | filtering in the mobile client, not the server; the list endpoint leaks what the item endpoint hides | response schemas + explicit writable-field allow-list after authorization |
| 4 | API4 Unrestricted resource consumption | X/Twitter 2022 phone-number lookup (Domoney case 7); OTP brute force cases | no rate limit on `/v2/auth/otp/verify`; unbounded `?page_size` | the limit on the login route, none on the *lookup* route that enumerates | gateway quotas + per-route budgets |
| 5 | API5 Function level (BFLA) | campus access-control system, admin functions open to student IDs (Domoney case 2) | `DELETE /v2/invoices/{id}` shares the GET handler's check | same URL, different verb; the admin prefix that `/v1` didn't have | authorization middleware with explicit "public"/role declaration per route |
| 6 | API6 Sensitive business flows | **Open:** find a primary case of excessive automated use of an otherwise authorized flow. Homakov's Starbucks gift-card race is a separate atomicity failure, not by itself an API6 example. | authorized quote → confirm requests repeated beyond the business limit | the UI limits the flow, while direct API calls do not | flow-level limits and abuse controls; idempotency only for duplicate confirmation |
| 7 | API7 SSRF | Shopify Exchange screenshot-service SSRF (HackerOne 2018, $25k) — verify from the disclosed report; its surface differs from Ledger's webhook | tenant webhook test fetches a caller-controlled URL | redirects and resolved addresses can cross the boundary after an initial URL check | one outbound policy, demonstrated with a fake resolver and transport |
| 8 | API8 Misconfiguration | home-router API (Domoney case 8); All in One SEO path-case bug (case 6) | permissive CORS, verbose errors, debug route left on | case-insensitive path matching bypasses the prefix check from ch. 5 | hardened defaults recorded in the OpenAPI spec; positive model |
| 9 | API9 Inventory (zombie APIs) | Optus 2022: a dormant internet-facing domain remained vulnerable after an earlier access-control error (ACMA court filing 2024) — verify the filing before prose | the unlisted staging host and still-serving `/v1` | "deprecated" is a label; retirement means it stops answering | inventory from code + gateway + traffic; route/host comparison as the test |
| 10 | API10 Unsafe consumption | **Open:** find a primary case where a consumer API mishandles data or redirects from an integrated service. The currently named SiriusXM/Hyundai cases do not establish one partner-to-consumer chain. If no distinct source survives the search, fold the response-consumption lesson into ch. 7 and revise the register rather than invent an incident. | the exchange-rate feed's JSON is trusted when updating invoice totals | the response schema, size, redirect or destination is trusted because the source is a partner | validate and bound the consumed response at the integration boundary |
| 11 | Where every route must pass | none — the payoff chapter | — | — | assembles the controls (loader, middleware, schema, gateway, egress, inventory, two-user test loop, monitoring) as *one* diagram of the service |
| 0 | Opening (short) | Coinbase 2022: a trade's source account did not match its order-book asset; both accounts belonged to the same user | — | — | introduces the question "which check did this request assume had already happened?" and the invented service in about 400 words |

Incident rule: each chapter's incident comes from the **original public disclosure** (researcher
write-up, vendor advisory, bug-bounty report, or filed regulator account), fetched into
`resources/incidents/NN/` and quote-gated as in Book 17. Before drafting, write down the exact
mechanism in that source and why it fits the chosen OWASP class; replace a candidate that fails
this check. Domoney's retelling is the pointer, not the source. Numbers in prose must appear in a
fetched source or be absent. The Ledger example is an analogy, never presented as the real
incident's implementation.

## 4. The chapter contract (1,600–2,100 words)

1. **The incident** (2–4 paragraphs): who, what request or sequence, which check failed, and the
   documented consequence. Include a number only when it earns its place and has a primary source.
2. **The class in one sentence**, with its OWASP name.
3. **Where the bug lives**: the vulnerable and fixed handler, configuration, or request flow,
   with the few details that make the difference real.
4. **Why the local fix is insufficient**: the structural reasons this check gets missed or
   bypassed, each tied to a concrete path in Ledger.
5. **The second path or step**: the worked Ledger example and its twist. Follow that class's
   actual repair sequence: trace the input, put the check at the right boundary, exercise the
   bypass, and verify coverage. Chapter 1's shared loader and two-user loop are one instance,
   not a template imposed on SSRF, resource consumption, or partner responses.
6. `<!--mission-->` **Exercise**: a short set of requests, handlers, configurations or traces
   suited to that chapter's mechanism. **Every exercise has one plausible decoy and one
   near-identical pair with opposite outcomes.** Show expected results and a worked answer;
   vary the form and length, not the discriminating test. If a chapter cannot construct the pair,
   its brief must say why before drafting and the panel must explicitly test that exception.
7. One-line source credit.

No first-person scenes, no "the book says", no chapter numbers of the source in the body (Book 11's
current brief). Every code sample in the chapter is a real file in `service/`. Every expected
result that the toy service can exhibit is asserted in `service/chNN_test.go`; evidence about a
real incident and operational controls outside the toy service receive separate source or fixture
checks.

## 5. Verification (the repo's design, per the book pattern)

- `service/`: one Go module and one cumulative service. `NewApp(mode)` selects a deliberately
  vulnerable or fixed implementation; the fixed branch retains every earlier repair. Use
  `net/http/httptest`, an in-memory store, a fake clock and a fake outbound transport where needed.
- `service/chNN_test.go`: table-driven cases assert the observed vulnerable response or state
  **and** the fixed response or state. Both modes must pass their assertions; a test must fail if
  the intended vulnerability disappears, if the repair fails, or if a supposedly safe decoy leaks.
  `verify.sh` runs `go test ./...` from inside `service/` and checks that every chapter's cases ran.
- Chapter 9 also checks a route/host inventory fixture against the router and gateway fixture;
  `httptest` alone cannot prove an unknown deployed host was discovered.
- `checks/claims/NN.tsv` + `resources/incidents/NN/`: every number and quote in the incident section.
- `CONTEXT.md`: decision record, cast and fixed numbers (the spine table), correction log.
- Static HTML chapters + index, GitHub Pages, as always.

## 6. Pipeline and gates

1. **Pilot**: ch. 1 (rewrite the essay's *method* onto the spine with a genuine BOLA disclosure
   and executable tests) and ch. 9 (zombie APIs — the essay's twist at full size). Main session
   drafts ch. 1; ch. 9 pitched by codex,
   drafted by the main session. Panel: `agy` gemini-3.8-flash-high + gemini-3.1-pro-high,
   codex sol consolidates. **The author reads both pilots before anything else is drafted.**
   The author's reading gate asks where attention dropped, whether the Chapter 9 reveal added a
   new idea, and whether the exercise made the missing check predictable. Model review does not
   answer those reader-experience questions.
2. Then chapters in batches of three: incident fetch + pitch by codex, draft by the main session
   (one at a time) or codex sol for well-specified ones, panel, revision by codex, read-through.
3. Ch. 11 last, written from the finished service.

Estimated prose size: about 21–23k words. Service and test size are provisional; the pilots will
establish a credible estimate. Do not compress a worked step to satisfy a line or word budget.

## 7. Decisions (author, 2026-09-23)

1. Title: **Security Rebook**.
2. Incidents for ch. 6/7/9/10: author left the pick to the editor. The source-and-class check
   above kept ch. 7/9 as candidates and reopened ch. 6/10; no incident is locked until its
   primary source is fetched into `resources/incidents/NN/`.
3. Language: **Go**.
4. Pilots: ch. 1 + ch. 9, editor's call.
