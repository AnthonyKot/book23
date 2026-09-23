# Drafting brief — one page for a fresh agent writing one chapter of Book 23

Read, in this order, before writing anything: this file; `CONTEXT.md` (cast, fixed-number table,
code conventions, voice); `GUIDANCE.md` §2 (the chapter test); `chapters/01-whos-asking.md` (the
voice and form reference — not a plot template); the chapter register in `PLAN.md` §3 for your
chapter's row; and the briefs of the chapters before yours in `briefs/`, for the state of the
service. Do not read the other chapters in full unless you need a specific object from them.

## What a chapter is

1,600–2,100 words of prose. One real incident, told from the original public disclosure, which you
fetch and read yourself (Domoney's *Defending APIs* is only the pointer). One OWASP API class,
named in one sentence. The bug shown on Ledger in a vulnerable handler and a fixed handler, with
the few details that decide whether the fix is real. Why the local fix is insufficient, tied to a
concrete path in Ledger. A second path or step where the same check is skipped again, using an
object or repair from an earlier chapter *inside the worked example*. An exercise after a
`<!--mission-->` line with a plausible decoy and a near-identical pair with opposite outcomes,
and a worked answer that traces input → check (or its absence) → response or state change. A
one-line source credit.

Every Go block is a `{{excerpt:chNN-name}}` placeholder; the service is built later from your
brief. Never present invented Go as executable or say tests pass. Put an HTML comment at the top
listing every incident fact the claims file must gate, with the source URL. If the primary record
does not support the assigned class, stop and write the brief with sourced alternatives; never
invent incident detail. Numbers only when the record has them.

Then write `briefs/NN.md`: mechanism sentence and class fit; Ledger objects reused; handlers and
mode behaviour needed; the exercise-case matrix (route, caller, request, expected vulnerable
outcome, expected fixed outcome); excerpt names used; proposed additions to CONTEXT's fixed table
(no look-alike numbers); claims needing source checks; anything unverified.

Write only `chapters/NN-slug.md` and `briefs/NN.md` in `/home/diablo/book23`. Do not commit. Do not edit any other file: a Codex review pass may be running in this tree at the same time on chapters 0–3 and CONTEXT.md.

## State of Ledger after chapters 0–4 (what you may rely on and must not contradict)

| After | Repair or fact in force |
|---|---|
| 0 | Ledger: invoicing API, tenants Cedar and Birch; Alice (Cedar), Ben (Birch), Dana (Cedar admin), Ops (platform team, gateway); `/v2` current, `/v1` older and still called by the mobile app; Go stdlib only |
| 1 | `LoadInvoiceFor(user, id)` is the only way a handler gets an invoice; 404 for missing and forbidden alike; `/v1` read and `/v1/invoices/{id}/pdf` go through it but are **not retired** (that is ch. 9); two-user test loop over every invoice route; refund quote `q-771` → confirm must use the invoice stored with the quote |
| 2 | `currentUser` resolves a bearer token through the session store (12-hour lifetime, fake clock); the mobile app key `mk_ledger_mobile_…` alone is not an identity; `/v1`'s old `X-User` header is ignored everywhere; tenant keys `ck_cedar_…` / `bk_birch_…` identify a tenant integration, not a person |
| 3 | Invoice 104 = £1,800.00, 205 = £640.00; stored `store.Invoice` carries customer contact fields plus Ledger-internal `CollectionsNote` and `Margin`; responses encode `ViewFor(user, inv)` (ordinary vs admin view), never the row; `store.Invoice.MarshalJSON` returns an error; PATCH decodes typed `InvoicePatch{Reference}` / `ProfilePatch{DisplayName}` with unknown fields rejected; `/v1` PDF template bound to the view |
| 4 | (CONTEXT register row still says "X phone lookup 2022"; the switch to Instagram 2019 is decided and is being recorded by the review pass.) OTP: 6 digits, 10-minute challenge, budget of 5 wrong attempts **per challenge** then locked; the per-IP limiter stays as a second, separate limit; `?limit=` capped at 50; 30 requests/minute per lookup route, applied at Ops's gateway; incident is Instagram 2019 (Muthiyah), X/Twitter 2022 is an aside |

Chapters 5–11 add to this table; a later chapter may not silently undo an earlier repair.

## Register for the next chapters (from PLAN §3; incidents still need the source-and-class gate)

- **5 — Same Door, Different Verb** (API5 function level): campus access-control system, 2022,
  admin functions reachable with student IDs (Domoney case 2; find the researcher's own write-up
  and the TechCrunch report). Ledger: `DELETE /v2/invoices/{id}` and the admin routes share the
  GET handler's check; same URL, different verb; the admin prefix `/v1` never had. Repair:
  authorization middleware with an explicit role or "public" declaration per route.
- **6 — Every Request Was Valid** (API6 sensitive business flows): **incident open**. Needs a
  primary case of excessive automated use of an authorised flow (not a race condition). Ledger:
  refund quote → confirm repeated beyond the business limit; the UI limits the flow, the API
  does not. Repair: flow-level limits and abuse controls; idempotency only for duplicate confirms.
- **7 — The Server That Fetched for You** (API7 SSRF): Shopify Exchange screenshot SSRF,
  HackerOne 2018, $25,000 — verify from the disclosed report; its surface differs from Ledger's.
  Ledger: the tenant webhook "test" fetches a caller-controlled URL; redirects and resolved
  addresses cross the boundary after the initial URL check. Repair: one outbound policy at one
  egress, demonstrated with a fake resolver and transport.
