# Lane A batch 2 implementation handover

The reviewed Chapters 00–10 provide the prose and source gates. This handover tracks the
remaining cumulative Ledger implementation, one chapter per run.

| Chapter | State | Next evidence |
|---|---|---|
| 01 | Complete for batch 2; refund/list cases and route coverage tested | Author reconciles schematic exercise wording |
| 02–08 | Reviewed prose and briefs; service pending | Per-chapter code, excerpts and tests |
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

Next: Chapter 02 sessions, with both-mode expiry tests and the 12-hour fake clock.
