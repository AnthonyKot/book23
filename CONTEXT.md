# CONTEXT — Security Rebook (Book 23)

Authority order: this file → PLAN.md → GUIDANCE.md → chapter briefs. Started 2026-09-23.

## 1. Decision record

| Date | Decision | Why |
|---|---|---|
| 2026-09-23 | Book grown from the Book 11 essay "The Lookup That Never Asks Who's Asking"; source backbone Domoney, *Defending APIs* (2024); chapter order = OWASP API Security Top 10:2023 | the author liked that essay most; its form (incident → fifteen-line bug → why structural → twist → decoy exercise) is the book |
| 2026-09-23 | Title: **Security Rebook** | author |
| 2026-09-23 | One running service, attacked chapter by chapter, with fixed names and numbers reused | Book 2's lesson the same day: bridges do not connect chapters, a running system does (GUIDANCE L1–L2) |
| 2026-09-23 | Language: **Go**, standard library only (`net/http`, `net/http/httptest`, `encoding/json`) | author; no framework so every check is visible in the handler; matches Book 13 |
| 2026-09-23 | Every chapter's code is real and its exercise table is a test; `verify.sh` runs vulnerable and fixed modes | GUIDANCE L9 |
| 2026-09-23 | Incidents come from primary disclosures, quote-gated; Domoney is the pointer | GUIDANCE L10; Book 17 method |
| 2026-09-23 | Pilots: ch. 1 (BOLA) and ch. 9 (zombie APIs); the author reads both before any other chapter is drafted | GUIDANCE L7 |
| 2026-09-23 | Book 17's incidents (Capital One, MongoDB 2017, xz, Adobe 2013, MOVEit, Strava, Colonial, NotPetya, Target, Equifax, goto fail, CrowdStrike) are off limits | no overlap between the two security books |
| 2026-09-23 | Editorial correction: Coinbase remains the opening incident, while ch. 1 needs a genuine cross-user BOLA case; Peloton 2021 is the current candidate | Coinbase's own retrospective describes a source-account/asset mismatch within one user's accounts, not another user's object |
| 2026-09-23 | Editorial correction: ch. 6 and ch. 10 incident choices remain open pending source-and-class fit | the proposed Starbucks race is not the same mechanism as API6's excessive access; the named SiriusXM/Hyundai cases do not establish the proposed API10 partner-to-consumer chain |

## 2. The service (the spine)

**Ledger**, a small invoicing API run by an invented company. It is a teaching example and says so
once, in chapter 0. No real company, product or person is implied.

### Cast

| Name | Role | Tenant |
|---|---|---|
| Alice | ordinary user | Cedar |
| Ben | ordinary user | Birch |
| Dana | tenant admin | Cedar |
| Ops | the platform team that owns the gateway and inventory | — |
| "the researcher" | the outside reporter in worked examples (never named, never given a biography) | — |

Only these people exist. Chapters never invent colleagues, meetings or quotes for them.

### Fixed numbers and names (GUIDANCE L4: no two may coincide or differ only by a suffix)

| Object | Value | First set | Reused by |
|---|---|---|---|
| Tenants | Cedar, Birch | ch. 0 | all |
| Invoices | 104 (Cedar), 205 (Birch) | ch. 1 | 3, 5, 6, 9 |
| Refund quote | quote `q-771` for invoice 104 | ch. 1 | 6 |
| Current API | `/v2` | ch. 0 | all |
| Older API | `/v1` (read route + `/v1/invoices/{id}/pdf`); protected in ch. 1, retired in ch. 9 | ch. 1 | 8, 9, 11 |
| Internal host | `ledger-staging.internal` (unlisted) | ch. 9 | 11 |
| Shared loader | `LoadInvoiceFor(user, id)` | ch. 1 | 3, 5, 6, 10 |
| Tenant API key | `ck_cedar_…` / `bk_birch_…` (prefix shows tenant; shown truncated) | ch. 2 | 10 |
| Session token lifetime | 12 hours | ch. 2 | 4 |
| OTP length / attempts | 6 digits, unlimited (vulnerable) → 5 attempts then lock (fixed) | ch. 4 | — |
| Page size | `?limit=` unbounded (vulnerable) → max 50 (fixed) | ch. 4 | — |
| Per-route budget | 30 requests/minute on lookup routes | ch. 4 | 11 |
| Webhook URL per tenant | `https://hooks.cedar.example/ledger` | ch. 7 | 11 |
| Egress allow-list | public IPs only, no redirects followed | ch. 7 | 11 |
| Partner feed | exchange rates from `rates.partner.example` | ch. 10 | 11 |
| Rate row Ledger trusts | `{"EUR": 0.92}` (vulnerable: written straight into invoice totals) | ch. 10 | — |
| Two-user test loop | every route serving invoices × {Alice→104, Alice→205, Ben→104} | ch. 1 | all |

Bounty amounts and dates belong to real incidents and live in `checks/claims/NN.tsv`, not here.

### Code conventions

- Go module `service/`. One package per chapter is wrong; one service, with the chapter's
  vulnerable and fixed handlers side by side and selected by `Mode` (`vulnerable`, `fixed`).
  The book shows the vulnerable handler, then the fixed one, exactly as they are in the file.
- Handlers are plain `http.HandlerFunc`; routing by `http.ServeMux` patterns (`GET /v2/invoices/{id}`).
- The store is an in-memory map seeded from `seed.go` with the fixed table above.
- A 404 for both missing and forbidden invoices, as in the essay; the chapter says why and that
  403 is also defensible.
- Tests: `chNN_test.go`, table-driven, with each executable exercise row recording the request,
  caller, observed vulnerable result and expected fixed result (status, body property, and any
  state change). Each test constructs `NewApp(Vulnerable)` and `NewApp(Fixed)` and asserts both
  results; both sets of assertions must pass. `verify.sh` runs `go test ./...` **from `service/`**
  and checks that chapter cases ran. It must not treat an arbitrary failing test run as proof
  that a vulnerability exists.
- The fixed mode accumulates earlier repairs. The vulnerable mode is a controlled counterexample,
  not a claim that every flaw coexisted at one moment in the fictional service's history.
- No external dependencies in the first pilot. Use opaque fixture tokens and the standard
  library for the teaching service; do not hand-roll JWT verification or present fixture auth,
  an in-memory rate limiter, or a toy egress policy as production-ready security. Inject a fake
  clock and fake outbound transport for deterministic, network-free tests.

## 3. Voice and form

- Address the reader as "you"; the incident is told in the past tense from the public record;
  the service in the present.
- One incident with a number, one bug class, one twist, one exercise per chapter (GUIDANCE L6).
- Headings are claims ("The route you fixed, and the one you forgot"), never labels.
- No first-person scenes, no "the book says", no chapter numbers of the source in the body.
- Every code block is a real file excerpt; no pseudo-code. The excerpt is at most ~20 lines; the
  rest is in `service/`.
- Every exercise verdict traces the relevant untrusted input or event, the check that runs (or
  does not), the request or sequence, and the observed response or state change (GUIDANCE L5).
- The exercise includes a plausible decoy and, where the mechanism supports it, a near-identical
  pair with opposite verdicts. Vary the exercise form to fit the class (GUIDANCE L8).
- 1,600–2,100 words; the pilot sets the number; ±15% after that.
- Real incidents: nothing that is not in a fetched primary source. If the record does not say
  how long the fix took, the chapter does not say either.

## 4. Chapter register (status)

| # | Title (working) | Class | Incident | Status |
|---|---|---|---|---|
| 0 | The Lookup That Never Asks | opener | Coinbase 2022 (source-account/asset mismatch) | planned |
| 1 | Who's Asking | API1 BOLA | Peloton 2021, candidate pending chapter source gate | **pilot** |
| 2 | One Key for Every Door | API2 authentication | BrewDog 2021 | planned |
| 3 | The Row You Didn't Mean to Send | API3 property level | shipping-company API (Domoney case 1), primary source to find | planned |
| 4 | Nobody Counted | API4 resource consumption | X phone lookup 2022 | planned |
| 5 | Same Door, Different Verb | API5 function level | campus access control 2022 | planned |
| 6 | Every Request Was Valid | API6 business flows | open: find a primary excessive-flow case | planned |
| 7 | The Server That Fetched for You | API7 SSRF | Shopify Exchange 2018 | planned |
| 8 | Left On | API8 misconfiguration | home router (Domoney case 8) / AIOSEO | planned |
| 9 | Deprecated Is a Label | API9 inventory | Optus 2022 | **pilot** |
| 10 | What You Swallowed | API10 unsafe consumption | open: find a primary downstream-consumption case | planned |
| 11 | Where Every Route Must Pass | payoff | — | planned last |

## 5. Correction log

- 2026-09-23: [Coinbase's retrospective](https://www.coinbase.com/blog/retrospective-recent-coinbase-bug-bounty-award)
  confirms a mismatch between the source account's asset and the trade's order book; it does
  not document cross-user invoice-style BOLA. The Book 11 essay is a model for form, not a
  source-accurate BOLA incident. [Jan Masters' Peloton disclosure](https://www.pentestpartners.com/security-blog/tour-de-peloton-exposed-user-data/)
  is a candidate because requiring login still let members see others' information; chapter 1
  must verify the exact endpoint and claim before using it.
- 2026-09-23: [OWASP's API6 definition](https://api-security.owasp.org/editions/2023/en/0xa6-unrestricted-access-to-sensitive-business-flows/)
  concerns excessive access to a sensitive flow; a race in gift-card transfers is a different
  failure. [Sam Curry's automotive write-up](https://samcurry.net/web-hackers-vs-the-auto-industry/)
  separates the Hyundai account takeover and SiriusXM key exposure; it does not establish the
  proposed combined API10 chain. Both chapter incident slots remain open.
