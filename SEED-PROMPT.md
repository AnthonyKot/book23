# Seed prompt — Book 23 pilots, two lanes

Work in `/home/diablo/book23`. Read `CONTEXT.md`, `PLAN.md`, `GUIDANCE.md`,
`docs/CH01-INCIDENT-CHOICE.md`, and `/home/diablo/book11/essays/defending-apis.md` first.
The eventual deliverable is a short opener and two reviewable pilots, Chapters 1 and 9. This
prompt separates the source/code/test handoff from the main session's prose. No other chapters
or site publication. The author reads both pilots before the rest of the book is drafted.

The reader can follow Go handlers and HTTP requests but is learning to trace trust decisions
across routes and request sequences. The chapters must turn on the exact missing check, not an
OWASP label. Keep the real incident, fictional Ledger service, and lesson distinct.

## Lane A — Codex preparation; stop at the handoff

1. **Clear the source gate.** Fetch and preserve the original public records for Coinbase,
   Peloton, USPS Informed Visibility, and Tinder, plus the filed account for Optus. The comparison
   in `docs/CH01-INCIDENT-CHOICE.md` is the starting assessment: write one sourced mechanism
   sentence for each of Peloton, USPS, and Tinder in the handoff, and confirm whether Peloton
   still earns Chapter 1. Record the URLs and claim-level evidence for chosen incidents in
   `checks/claims/00.tsv`, `01.tsv`, and `09.tsv`. Coinbase's source-account/asset mismatch is
   not cross-user BOLA. For Peloton, identify `POST /stats/workouts/details` and the partial fix
   that required login but left cross-member exposure. For Optus, use the filed account of the
   dormant domain rather than a retelling. If a candidate cannot carry its assigned class,
   recommend a sourced replacement before prose; never bridge the gap with invented details.
   Keep quotations short and attributed.
2. **Build only the Ledger surface these pilots need.** Create one Go 1.22 module in `service/`,
   standard library only, with Alice/Cedar, Ben/Birch, Dana, invoices 104/205, the current `/v2`
   route, the still-serving `/v1` read and PDF routes, and the Chapter 9 staging-host fixture.
   `NewApp(Vulnerable)` and `NewApp(Fixed)` should expose a controlled contrast; fixed mode
   retains Chapter 1's loader check when Chapter 9 is added. Chapter 1 protects `/v1` without
   claiming to retire it. Chapter 9 discovers the unlisted staging host, reconciles code,
   gateway and traffic fixtures, and retires `/v1`. Keep checks deterministic and tests offline.
3. **Make tests prove both sides.** In `service/ch01_test.go` and `ch09_test.go`, assert the
   vulnerable and fixed response or state for each proposed exercise case. Each pilot's case
   table must include a plausible decoy and a near-identical pair with opposite outcomes. Both
   modes' assertions must pass; an unrelated failing test is never proof of an attack. From
   `service/`, `go test ./...` must pass. Set up `verify.sh` to check pilot test groups, links,
   chapter structure and claim files once the prose exists. Document what the toy service cannot
   prove about a real deployment.

Hand off the sources and claim ledger, the four mechanism sentences (Coinbase plus the three
Chapter 1 options), Optus evidence, the exact Go paths, a concise exercise-case matrix, the
passing test command, and unresolved claims. Commit the preparation as a coherent local
checkpoint, then **stop**. Do not draft the opener or chapters in this lane. Do not push.

## Lane B — main writing session, after the handoff

1. **Write the opener and Chapter 1.** The opener takes about 400 words and accurately names
   Coinbase's missing account/asset consistency check. Chapter 1 uses the selected Peloton
   record (or the sourced backup if the gate rejects it), then traces Ledger's separate
   Alice-to-Ben invoice bug through a real Go handler, `LoadInvoiceFor`, the still-serving
   `/v1` door, and a two-user exercise. Keep one visible error, why it recurs across routes,
   a plausible decoy, and a near-identical pair with opposite outcomes. The answer traces
   request → lookup → check → response and explains the contrast. Every printed Go excerpt
   must exactly match `service/`.
2. **Write Chapter 9.** Add a new discovery: the unlisted host and the mismatch among code
   routes, gateway configuration, and observed traffic. Show that `/v1` was protected in
   Chapter 1 yet still needed inventory and actual retirement. Use a testable inventory
   exercise with its own plausible decoy and near-identical pair with opposite outcomes.
3. **Reconcile prose and tests.** If writing changes an exercise case, update the corresponding
   test. Run `go test ./...` from `service/` and the complete `verify.sh` after prose exists.
   Check every incident claim against its source and every printed code excerpt against its
   file. Read each chapter with code folded out: can a reader name the missing check, predict
   the decoy and contrasting pair, and name the next door to inspect? Read Chapters 1 and 9
   together: does Chapter 9 use a specific object or repair from Chapter 1 and add a new idea?

Commit the opener and each pilot as coherent local checkpoints. Record remaining source or
mechanism uncertainty and stop for the author's reading. Do not push or publish.
