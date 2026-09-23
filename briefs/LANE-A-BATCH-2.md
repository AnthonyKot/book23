# Lane A, batch 2 — service code, tests, claims for chapters 1–7

For the author to run in Codex CLI from `~/book23`, after the review pass on chapters 0–3 has
committed or been abandoned (check `git status` and `git log` first):

    codex exec -m gpt-5.6-sol -s workspace-write -C ~/book23 "$(cat briefs/LANE-A-BATCH-2.md)" < /dev/null

---

You are Lane A of Book 23, "Security Rebook", in this directory: sources, service code and tests.
You do not write chapter prose. Read first: `CONTEXT.md` (the fixed table is binding),
`GUIDANCE.md` §2 and §5, `docs/HANDOVER.md`, `briefs/DRAFTING-BRIEF.md` (its "State of Ledger"
table is the cumulative design), `service/README.md`, the existing `service/*.go` and tests, then
`briefs/02.md` through `briefs/07.md` and the chapters they belong to. Before starting, run
`git status`; another process may have committed since this brief was written. Do not push.

## Ground rules

- Go 1.22, standard library only, one package `ledger` in `service/`. `gofmt` clean. Tests offline
  and deterministic (fake clock, fake resolver and transport for chapter 7).
- **Cumulative states.** Follow the pattern of `NewApp` and `NewChapter9App`: add
  `NewChapterNApp(mode)` for N = 2…7. Its `Vulnerable` side has every earlier chapter's repair in
  force and only chapter N's bug open; its `Fixed` side adds chapter N's repair. A later chapter
  never undoes an earlier repair. If two chapters' designs conflict, the later brief loses; record
  the conflict in `docs/HANDOVER-2.md` rather than resolving it silently.
- **Excerpts.** Every `{{excerpt:chNN-name}}` placeholder in `chapters/` names a block you must
  wrap in `// excerpt: chNN-name` … `// end excerpt` markers around the exact source. Then replace
  the placeholder in the chapter with the fenced block, verbatim, in the form chapter 1 uses
  (a ```go fence whose first line is `// chNN-name`). That is the only prose edit you make. If a
  brief's block cannot be built as described (for example chapter 6's per-user counter, which is
  the rejected local fix and is not wired into any build), build it as an unexported helper with a
  direct test, mark it, and note in `docs/HANDOVER-2.md` that it is unwired.
- **Tests prove both sides.** Every row of every chapter's exercise table (in the chapter and in
  its brief's matrix) gets a test asserting the vulnerable and the fixed outcome, in
  `service/chNN_test.go`, grouped so `verify.sh` can require it. Sequences (chapter 6) are tests
  of sequences. If a table row cannot be asserted, say which and why in the handover; do not
  weaken the assertion to make it pass.
- **Fixed values.** Amounts are pence (`104` = 180000, `205` = 64000). Refund allowance £1,000 per
  tenant per day = 100000 pence. Quote `q-771` is Alice's first quote on 104. Session lifetime 12
  hours. OTP six digits, five wrong attempts per challenge. `?limit=` max 50. Rate limit 30 per
  minute per lookup route at the gateway. Webhook `https://hooks.cedar.example/ledger`. No new
  number may look like an existing one; propose additions in the handover, never invent silently.

## Work, in order, one commit per chapter (`service: chNN`)

1. **Chapter 1 gaps.** The chapter's exercise has two cases with no handler behind them: the
   refund `quote` → `confirm` pair (case D: confirm must refund the invoice stored with the quote
   and ignore the body's `invoice_id`) and the client-filtered list `GET /v2/invoices?tenant=X`
   (case E: fixed build filters on the caller's tenant). Add both to `NewApp` in both modes, with
   tests. Also add the test chapter 1 now promises: compare the routes the app registers with the
   routes the two-user loop covers, and fail if any invoice-serving route is missing from the loop.
   Chapter 6 builds on this refund flow, so match `briefs/06.md`'s shapes where they overlap.
2. **Chapters 2, 3, 4** from `briefs/02.md`–`04.md`: session store and `mk_ledger_mobile_…` key;
   `ViewFor`, typed patches, `MarshalJSON` error on the raw row; OTP challenge with per-challenge
   budget, per-IP limiter as a separate limit, `?limit=` cap. Create `checks/claims/02.tsv`,
   `03.tsv`, `04.tsv` from the chapters' header comments with locators into the archived sources
   (02 and 03 are already in `resources/incidents/`; fetch and archive 04's Instagram/Muthiyah
   write-up and add its `SOURCES.tsv` row with checksum). A fact the archive does not support gets
   a row marked UNSUPPORTED and a line in the handover; do not edit the prose to fix it.
3. **Chapters 5, 6, 7** from `briefs/05.md`–`07.md`: declared access levels and middleware (403);
   refund allowance in `store.ConfirmRefund`, approval route; the `egress` client with fake
   resolver and transport, webhook test route, PDF logo fetch through it. Same claims and archive
   work for 05 (DEF CON slides PDF + TechCrunch), 06 (FTC complaint PDF, stipulated order, release)
   and 07 (HackerOne report JSON, Shopify blog).
4. **verify.sh**: require each chapter's test group; check that no `{{excerpt:` remains in any
   chapter whose service code exists; keep the HTML checks pending until a site exists.
5. **docs/HANDOVER-2.md**: what was built, what each fixture cannot prove, every design conflict
   and every table row not asserted, and proposed CONTEXT additions. Run `cd service && go test
   ./...`, `./verify.sh`, `git diff --check` before the final commit.

Do not edit `PLAN.md`, `GUIDANCE.md`, `CONTEXT.md` (propose in the handover), any chapter beyond
placeholder filling, or any file under `review/`. Chapters 8–11 are not in this batch.
