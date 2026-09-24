# Lane A, batch 2 — build the reviewed Ledger service

Run from `~/book23` after the chapter reviews. This is a resumable implementation prompt:
**one chapter per invocation**, in the order below. Re-run the same prompt for the next
chapter. Do not push.

    codex exec -m gpt-5.6-sol -s workspace-write -C ~/book23 "$(cat briefs/LANE-A-BATCH-2.md)" < /dev/null

The 00–10 prose review is complete. Claims ledgers and primary archives for Chapters 02–08
and 10 already exist; verify and use them, do not fetch or recreate them by default. Chapter
09 already has a Go pilot and tests, but its current `NewChapter9App` applies Chapter 01's
repair and Chapter 09's retirement only. It is **not yet cumulative through Chapters 02–08**.
The service presently has Chapter 01 and 09 code; most other chapter excerpts remain
placeholders. Read the current tree rather than assuming this status is still current.

You are Lane A: service code, tests, exact code excerpts, and the implementation handover.
Do not write or editorially revise chapter prose. Read `CONTEXT.md` (fixed values are
binding), `GUIDANCE.md` §2 and §5, `briefs/DRAFTING-BRIEF.md` (State of Ledger),
`docs/HANDOVER.md`, `service/README.md`, the relevant chapter and brief, the existing Go
code and tests, and the review report for that chapter in `review/`. Start every invocation
with `git status -sb` and `git log --oneline -8`; preserve unrelated work. If
`docs/HANDOVER-2.md` exists, read it to locate the next incomplete chapter. If it does not,
create it with a completion table and the current pilot state. Check completion against code,
tests, excerpts and commits, not a status line alone.

## Architecture contract

- Go 1.22 standard library; package `ledger`; `gofmt`; offline deterministic tests.
  Inject clocks, and for Chapter 07 inject DNS/dial and HTTP response behavior. No real
  network in tests.
- Keep `NewApp(mode)` as the Chapter 01 lesson state. Add
  `NewChapterNApp(mode)` for N = 2–8 and 10. For Chapter N, vulnerable mode includes every
  repair from Chapters 01 through N−1, leaving only N's bug open; fixed mode adds N's
  repair. Chapter 09 must be **integrated after Chapter 08**: refactor
  `NewChapter9App(mode)` so vulnerable mode includes Chapters 01–08's repairs and the
  still-serving v1/staging surface, while fixed mode retires v1/staging. Preserve its
  current exercise and inventory tests. Chapter 10 begins from Chapter 09 fixed.
- The current Chapter 09 pilot is a partial implementation, despite its comment saying
  "cumulative." Do not treat its Chapter 01-only fixture as proof that Chapters 02–08 are
  present. Make the additional routes, settings and handler repairs explicit in its
  registered-route inventory. Keep the Chapter 09 printed excerpt blocks byte-for-byte
  aligned with their marked Go blocks; use wrappers or unmarked assembly code where
  possible. If changing a marked block is unavoidable, record the conflict in the
  handover and stop before silently drifting the printed chapter.
- `Route` registration, authentication, views, authorization, limits, egress and settings
  should compose through one implementation path. Avoid independent chapter forks that
  pass their own tests while omitting earlier repairs. Exercise tests should probe at least
  one earlier repaired behavior in each later stage.
- Chapter 04's 30/min email-lookup budget is at the gateway **and in the shared handler**,
  including the v1/v2 lookup paths. Chapter 09's staging host can miss the gateway copy,
  not the shared-handler limit. Invoice-by-ID is a different route. Preserve this review
  decision when composing the stages.
- Fixed values come from `CONTEXT.md`: invoices 104/205 are 180000/64000 pence;
  tenant refund allowance is 100000 pence/day; quote `q-771` is the first quote on 104;
  the EUR invoice is 412 (€300.00 → 27600 pence at 0.92), with its recorded `FXRate`.
  Do not change existing values or invent new canonical ones silently. Record any needed
  addition in `docs/HANDOVER-2.md`; use an isolated test-only value if it is only a
  fixture.

## Next-chapter order

Implement exactly the first incomplete entry, then stop and report. Commit it with
`service: chNN ...`. The next invocation continues at the following entry.

1. **01 — close the pilot exercise gaps.** Implement the refund quote → confirm case
   and `GET /v2/invoices?tenant=X` in both modes. Chapter 01's exercise is schematic:
   its quote body omits `amount`, and its worked answer both says a mismatched confirm is
   rejected and says the supplied `invoice_id` is ignored. Use Chapter 06's concrete
   quote body `{invoice_id, amount}` with its existing £400 example; the first quote on
   104 is `q-771`. The vulnerable confirm may accept `{quote_id, invoice_id}` and act on
   the fresh ID. Fixed confirm accepts `{quote_id}`; an extra `invoice_id` is rejected
   with 400 and no refund, while a valid confirm uses the invoice stored with the quote.
   Test Alice's `q-771` plus injected 205 in both modes and record the exercise's
   schematic-versus-wire discrepancy in the handover for the author. For the list case,
   use the exercise's `tenant=birch` query (tenant-name matching is case-insensitive):
   vulnerable Alice receives 205; fixed returns 200 with an empty list.
   Add the promised registered-invoice-route versus two-user-loop coverage test.
   Preserve existing Chapter 01 and 09 excerpt text and tests.
2. **02 — sessions.** Follow `briefs/02.md`: 12-hour session store, current user from
   session token, mobile app key `mk_ledger_mobile_…` alone not an identity, old v1
   `X-User` ignored. Test both modes and expiry with a fake clock.
3. **03 — properties and response views.** Follow `briefs/03.md`: `ViewFor`, typed
   patches with unknown fields rejected, a raw `Invoice.MarshalJSON` error, and PDF
   bound to the view. Preserve Chapter 01's printed handlers as its historical lesson;
   later stages may use a new checked response path.
4. **04 — resource consumption.** Follow `briefs/04.md`: per-challenge OTP budget,
   separate per-IP limiter, page limit 50, and the 30/min lookup budget at both the
   gateway and shared handler.
5. **05 — function-level access.** Follow `briefs/05.md`: declared
   `Public`/`User`/`TenantAdmin` route levels, missing declaration fails registration,
   tenant scope before role denial, same-tenant wrong role → 403. Do not reintroduce
   the discarded /admin prefix guard.
6. **06 — business-flow allowance.** Follow `briefs/06.md`: repeatable quotes,
   confirm idempotency, tenant daily allowance in `store.ConfirmRefund`, parked quote
   and Dana approval, plus the per-user counter only as the rejected local-fix example.
7. **07 — outbound request boundary.** Follow `briefs/07.md`: one `egress` client
   with injected resolver/dial callback and canned RoundTripper; public address only,
   pinned dial target, no redirects; webhook test and PDF logo fetch use it.
8. **08 — production settings.** Follow `briefs/08.md`: one `Settings` literal per
   environment, route-table walk against `PublicRoutes`, debug-only denial reason,
   opaque fixed 404 and the admin health route.
9. **09 — cumulative integration.** Incorporate Chapters 02–08 into the existing
   `NewChapter9App`. Keep the v1/staging vulnerable-versus-fixed exercise, inventory
   reconciliation, and marked excerpt equality. Test the earlier session, view,
   access, lookup-budget and settings repairs through Chapter 09 as well.
10. **10 — partner response boundary.** Follow `briefs/10.md`: rate poller through
    `egress`, typed `rateRow{EUR}`, unknown fields rejected, 0.70–1.10 band,
    last-good rate and Ops page, invoice `FXRate`. Convert requested EUR to integer
    pence at the invoice's recorded rate **before** Chapter 06's remaining-balance
    check; store quote pence for confirm and the tenant allowance. Legacy EUR invoices
    without `FXRate` get 409 `rate_reconciliation_required`. Test the 0.95 redirect
    contrast and the full €300 refund example.

## Finish each invocation

- Every exercise-table row for the selected chapter gets a vulnerable and fixed assertion
  in `service/chNN_test.go`, including state/body where status alone hides the difference.
  Sequence rows are tested as sequences. The teammate's hypothetical fix is explanatory;
  test the actual modes. If a row cannot be asserted, name it and why in the handover.
- For each built chapter, wrap each requested source block in unique
  `// excerpt: chNN-name` / `// end excerpt` markers, and replace that chapter's
  `{{excerpt:...}}` placeholders with the exact fenced Go source. Do not change other
  prose. When code and tests are complete, add the one-line built status to its brief
  and shrink its header claims comment to point at the existing claims TSV. Do not retire
  a brief if the implementation is incomplete.
- Update `verify.sh` incrementally: require the built chapter's test group, validate
  all existing claims ledgers against their archived sources, and reject placeholders
  in a chapter marked built. Unbuilt chapters may still have placeholders. Keep
  rendered-page checks compatible with docs that have not yet been rebuilt.
- Update `docs/HANDOVER-2.md` after the selected chapter: completed state, tests and
  excerpt names, fixture limits, any design conflict or missing exercise assertion,
  and the next chapter. Run `cd service && go test ./...`, `./verify.sh` and
  `git diff --check`; inspect the staged diff; commit only this chapter's work.
- After Chapter 10 alone, run `node site/build.mjs`, verify rendered pages have the
  filled excerpts, and commit the generated `docs/` in a separate site commit.
  Chapter 11 is outside this batch. Never push.

Do not edit `PLAN.md`, `GUIDANCE.md`, `CONTEXT.md`, or files under `review/`.
If a brief contradicts an earlier established repair, preserve the earlier repair and
record the conflict in the handover. If a required example cannot be implemented
without changing an existing fixed value or a printed excerpt, stop that chapter with
a concrete BLOCK and options; do not make an unreviewed prose change.
