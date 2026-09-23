# Book 23 — plan (2026-09-23, draft for the author's decision)

Working title candidates: **Who's Asking** · *Every Door to the Table* · *The Lookup That Never Asks*.

## 1. What this book is

A book of API break-ins, one per chapter, grown from the Book 11 essay the author liked most
("The Lookup That Never Asks Who's Asking", `~/book11/essays/defending-apis.md`). The source
backbone is Colin Domoney, *Defending APIs* (Packt, 2024), already extracted in
`~/book11/workspace/defending-apis/` (95k words, page boundaries kept; corpus record in
`~/book11/corpus/defending-apis/`). Its spine is the OWASP API Security Top 10 (2023), which
Domoney's chapters 3, 7 and 9 follow.

The essay worked for five reasons, and each becomes a rule:

1. **A real incident with a number.** Coinbase, one edited account ID, $250,000. Not a taxonomy.
2. **The bug fits in fifteen lines**, shown vulnerable then fixed, with the three details that
   decide whether the fix is real.
3. **"If the fix is one line, why is it number one?"** The structural reasons the one line goes
   missing (auth is global, authz is per request; step two trusts step one; tests use one user).
4. **A worked example with a twist**: the route you fixed and the one you forgot (v1, the PDF
   export, the list filter). The reader learns the *question* ("which routes reach this data?"),
   not the patch.
5. **An exercise with decoys and an answer key**, doable without a running system, ending on the
   trap sentence ("If you marked D safe... that's the mistake that paid out $250,000").

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
- Surfaces: `/v2` (current), `/v1` (the zombie), a mobile client, a partner/third-party feed the
  service *consumes* (exchange rates), an API gateway in front.
- Implementation: FastAPI-style Python, the same idiom as the essay, small enough to read.

Rules of the spine (as in `~/book2-spine/docs/SPINE-BRIEF.md`): fixed names and numbers set once and
reused; later chapters may say "the loader from chapter 1" and mean a specific function; the
service's state accumulates (the v1 route retired in ch. 1 stays retired; the shared loader built in
ch. 1 is what ch. 3's property filter hangs off; the gateway inventory from ch. 9 is what ch. 11
monitors). Correctness beats the spine: where a class of bug doesn't fit billing, the chapter says
so and uses the smallest honest extension.

## 3. Chapter register (OWASP API Top 10:2023 order; incidents to be verified against primary disclosures before drafting)

| # | Class | Real incident (candidate) | The one-line bug on the service | The twist (door you forgot) | Choke point built |
|---|---|---|---|---|---|
| 1 | API1 BOLA | Coinbase 2022 ($250k bounty) — already written | `db.invoices.get(id)` never asks whose | `/v1/invoices/{id}/pdf`; refund confirm trusts body | `load_invoice_for(user, id)` |
| 2 | API2 Broken authentication | BrewDog app, hard-coded bearer token (Pen Test Partners 2021; Domoney case 3) | one shared token in the mobile binary; reset flow that returns the token | rotating the token ≠ fixing the design; the reset endpoint is the second door | per-user tokens, reset that never echoes the secret |
| 3 | API3 Object property level (excessive exposure + mass assignment) | shipping-company API returning full recipient details (Domoney case 1); Peloton 2021 profile API | handler returns the ORM row; PATCH accepts `is_admin` | filtering in the mobile client, not the server; the list endpoint leaks what the item endpoint hides | response schemas + explicit writable-field allow-list on the loader |
| 4 | API4 Unrestricted resource consumption | X/Twitter 2022 phone-number lookup (Domoney case 7); OTP brute force cases | no rate limit on `/v2/auth/otp/verify`; unbounded `?page_size` | the limit on the login route, none on the *lookup* route that enumerates | gateway quotas + per-route budgets |
| 5 | API5 Function level (BFLA) | campus access-control system, admin functions open to student IDs (Domoney case 2) | `DELETE /v2/invoices/{id}` shares the GET handler's check | same URL, different verb; the admin prefix that `/v1` didn't have | authorization middleware with explicit "public"/role declaration per route |
| 6 | API6 Sensitive business flows | ticket/sneaker-bot style abuse — pick from the public record | every request valid, the *sequence* is the attack (quote → confirm ×1000) | the flow is protected on the web UI, not on the API the UI calls | flow-level controls: idempotency, step tokens, abuse detection |
| 7 | API7 SSRF | webhook-URL SSRF from the public record (not Capital One — Book 17's) | tenant webhook URL fetched by the server, no allow-list | the "test webhook" button; redirects; the PDF renderer fetching images | outbound allow-list + resolver checks at one egress |
| 8 | API8 Misconfiguration | home-router API (Domoney case 8); All in One SEO path-case bug (case 6) | permissive CORS, verbose errors, debug route left on | case-insensitive path matching bypasses the prefix check from ch. 5 | hardened defaults recorded in the OpenAPI spec; positive model |
| 9 | API9 Inventory (zombie APIs) | Optus 2022 (unauthenticated endpoint on a forgotten surface, 9.8M records) — verify | the door nobody listed: staging host, `/v1`, `/internal` | "deprecated" is a label; retirement means it stops answering | inventory from code + gateway + traffic; Kiterunner-style enumeration as the test |
| 10 | API10 Unsafe consumption | Sam Curry's vehicle-telematics chain (Domoney case 9) or the smart-scale case (case 10) — verify which fits | the exchange-rate feed's JSON written straight into invoices | trusting the upstream's redirect/TLS/ schema; the partner's BOLA becomes yours | validate what you consume like input you were sent |
| 11 | Where every route must pass | none — the payoff chapter | — | — | assembles the eleven choke points (loader, middleware, schema, gateway, egress, inventory, two-user test loop, monitoring) as *one* diagram of the service |
| 0 | Opening (short) | the exchange story retold in two paragraphs | — | — | introduces the service and the cast in 400 words |

Incident rule: each chapter's incident comes from the **original public disclosure** (researcher
write-up, vendor advisory, bug-bounty report), fetched into `resources/incidents/NN/` and
quote-gated as in Book 17; Domoney's retelling is the pointer, not the source. Numbers in prose must
appear in a fetched source or be absent.

## 4. The chapter contract (1,600–2,100 words)

1. **The incident** (2–4 paragraphs): who, what request, what one edit, what it cost. A number.
2. **The class in one sentence**, with its OWASP name.
3. **Where the bug lives**: vulnerable handler, fixed handler, "three details that matter".
4. **If the fix is that small, why is it on the list?** 3–4 structural reasons, each one sentence
   the reader can test against their own codebase.
5. **The route you fixed and the one you forgot**: the worked example on the service, with the
   twist; then the numbered steps (find every door → move the check to the choke point → test with
   two users / two tenants → retire or record).
6. `<!--mission-->` **Exercise**: five handlers, at least one safe-looking decoy, a table of expected
   results, answer key, the trap sentence last.
7. One-line source credit.

No first-person scenes, no "the book says", no chapter numbers of the source in the body (Book 11's
current brief). Every code sample in the chapter is a real file in `service/` and every expected
result in the exercise table is a test in `tests/`.

## 5. Verification (the repo's design, per the book pattern)

- `service/`: the FastAPI app, with `vulnerable/` and `fixed/` variants per chapter selected by a
  flag, so the two-user tests can be shown failing and passing.
- `tests/test_chNN.py`: the chapter's exercise table, literally (route, caller, resource, expected).
  `verify.sh` runs them in both modes and fails if a "vulnerable" test *passes* or a "fixed" one fails.
- `checks/claims/NN.tsv` + `resources/incidents/NN/`: every number and quote in the incident section.
- `CONTEXT.md`: decision record, cast and fixed numbers (the spine table), correction log.
- Static HTML chapters + index, GitHub Pages, as always.

## 6. Pipeline and gates

1. **Pilot**: ch. 1 (rewrite the essay onto the spine with executable tests) and ch. 9 (zombie
   APIs — the essay's twist at full size). Main session drafts ch. 1; ch. 9 pitched by codex,
   drafted by the main session. Panel: `agy` gemini-3.8-flash-high + gemini-3.1-pro-high,
   codex sol consolidates. **The author reads both pilots before anything else is drafted.**
2. Then chapters in batches of three: incident fetch + pitch by codex, draft by the main session
   (one at a time) or codex sol for well-specified ones, panel, revision by codex, read-through.
3. Ch. 11 last, written from the finished service.

Estimated size: 12 chapters × ~1,900 words ≈ 23k words plus a ~1,200-line service.

## 7. Decisions needed from the author

1. Title (three candidates above, or another).
2. Incidents to confirm or swap for chapters 6, 7, 9, 10 (the ones marked *pick/verify*).
3. Python/FastAPI as the single language for the whole book (the essay's idiom) — or Go, given
   Book 13?
4. Pilot pair: ch. 1 + ch. 9 as proposed?
