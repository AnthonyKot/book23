# Lane A batch 2 implementation handover

The reviewed Chapters 00–10 provide the prose and source gates. This handover tracks the
remaining cumulative Ledger implementation, one chapter per run.

| Chapter | State | Next evidence |
|---|---|---|
| 01 | Complete for batch 2; refund/list cases and route coverage tested | Author reconciles schematic exercise wording |
| 02 | Complete for batch 2; session identity and exercise tested | Author reviews identity-probe wording |
| 03 | Complete for batch 2; views, patches, and guard tested | Author reconciles mixed PATCH wording |
| 04–08 | Reviewed prose and briefs; service pending | Per-chapter code, excerpts and tests |
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

Next: Chapter 04 resource limits. Chapter 09 still needs cumulative integration after Chapter 08.
Do not push this implementation branch from Lane A.
