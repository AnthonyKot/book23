# Review checkpoint — 2026-09-23

## Active pass: Chapters 04–07

Prompt: `review/IMPROVE-PROMPT-04-07.md`. At the start of this pass `git status --short` was clean
and `HEAD` was `a47f054`; the Chapter 00–03 review is already complete. Read the authority files,
the State of Ledger table, the Chapter 04–07 drafts and briefs, and the existing service tests.
Chapters 04–07 have no archived incident sources, claims ledgers, Go handlers or test files yet;
their excerpt markers are intentionally provisional. No source has been fetched in this pass yet.

Initial issues to verify against primary records and the implementation briefs: Chapter 04's
Instagram figures and Twitter aside need source gating, and its lookup-route exercise must make
the different counter outcomes observable. Chapter 05's brief says case F is 200/403 while its
chapter says 404/404; the route and role-check ordering also needs one consistent specification.
Chapter 06 must retain the complaint-versus-admission distinction and show a tenant-wide refund
allowance that cannot be bypassed with the integration key. Chapter 07's HackerOne and Shopify
facts need archiving; the proposed fake resolver/transport must specify the dialed destination so
its redirect and DNS contrasts can be tested without network access.

Next: fetch each chapter's named primary records, register hashes, create located claims files,
then write ranked findings before sentence-level edits. Work 04 → 07, committing each chapter as
`review: chNN`; update this section after each. Run the service Go tests, `./verify.sh`, and
`git diff --check`. Do not build service code, rebuild the site, push or publish in this pass.

## Chapter 00–03 pass complete

The detailed findings, edits, verdicts and deferred work are in `review/codex-2026-09-23.md`.
This resumed pass committed Chapter 00 as `6921af3`, Chapter 01 as `a281f0c`, and Chapter 02
as `fc75425`; Chapter 03 is the final chapter in this pass. Other commits on
`main` added the site and further exploratory material during this review; they are not part of
the Chapter 00–03 review commits. Nothing from this pass was pushed.

Chapter 03: the DPD primary article and its archived JSON screenshot now have eight located
claim rows in `checks/claims/03.tsv`. The screenshot is registered in `SOURCES.tsv` with SHA-256
`094599f5a3600da6d5b823b45e869f876ef2d673f3adb9c7feecfd33c29b9517`. The prose now
uses the dated disclosure timeline, acknowledges that the example email field is null, and
separates DPD's proven weak postcode gate from the property-policy question that Ledger models.
The Chapter 3 Go excerpts and both-mode tests remain unbuilt by design of `IMPROVE-PROMPT.md`.
`go test ./...` from `service/` and `./verify.sh` passed; `verify.sh` checked 11 chapter pages
and reported 31 code blocks pending. Before publication, settle the DPD/API3 class-fit question
and implement/test the chapter's six exercise cases. Do not infer Chapter 3 coverage from the
existing Chapter 1/9 tests.

## Resumed state (after `9db92b4`)

On resumption, `git status --short` showed only untracked `briefs/08.md`,
`chapters/08-left-on.md`, and `chapters/10-what-you-swallowed.md`, which belong to other work and
must be left alone. The intervening commits already corrected Chapter 00's Tree of Alpha sequence,
added claims 00-08/00-09, and registered the three archived sources in `SOURCES.tsv`. Chapter 1's
compiler and v3 sentences, the Chapter 2/3 premature test-file claims, CONTEXT's fixed values,
and the Go fixture amounts were also updated. The historical findings below record what this
checkpoint originally found; check this resumed-state section before acting on them.

Chapter 00 reviewed in the resumed pass: the only new prose edit narrows an absolute claim about
which checks passed. The Tree of Alpha correction and claim rows had already been committed by
another pass. Chapter 01 is next; its Peloton field attribution and executable-exercise claims
remain to inspect.

Chapter 01 reviewed in the resumed pass: source attribution and three Go excerpts corrected;
the pilot Go tests and `verify.sh` pass. Its refund and client-filter exercise cases still lack
handlers and both-mode tests, so the chapter remains provisional. Chapter 02 is next: create
`checks/claims/02.tsv` from the archived BrewDog disclosure before editing its prose.

Chapter 02 reviewed in the resumed pass: `checks/claims/02.tsv` now gates the BrewDog incident;
prose distinguishes possession of the static app key from a verified person or app build.
The A/C exercise pair works as a reasoning contrast, but its service handlers and tests remain
unbuilt. The D/E identity probe is now described more honestly and needs a concrete person-scoped
route in implementation. Chapter 03 is next: create its primary-source claim ledger, then
correct the DPD timing and property-level overstatements.

Resume in `/home/diablo/book23` using `review/IMPROVE-PROMPT.md`. The active prompt covers
chapters 00–03 only; Chapter 04 is outside this pass. Before this review, `main` was at
`ab1450a` and both worktrees were clean. No chapter prose, service code, tests, claims ledger,
or plan file has been changed by this review yet. No tests have been run in this pass.
For a later session, paste the contents of `review/RESUME-PROMPT.md`.

## Source work completed

- Read `CONTEXT.md`, `PLAN.md`, `GUIDANCE.md`, `docs/HANDOVER.md`, the drafting briefs,
  chapters 00–03, and the Chapter 1 test and service code.
- Fetched the original BrewDog and DPD researcher disclosures to
  `resources/incidents/02/brewdog-primary.html` and
  `resources/incidents/03/dpd-primary.html`.
- Found the researcher's original Coinbase thread reproduced in a Thread Reader archive at
  `resources/incidents/00/tree-of-alpha-thread.html`. Its source is
  <https://threadreaderapp.com/thread/1495014902582362112.html>, reproducing
  <https://twitter.com/Tree_of_Alpha/status/1495014902582362112>.
- SHA-256: Tree thread `204f3f151f592bd09e3e48dd96f36600a0c9ae1fab25d242019cb67bb88cee0a`;
  BrewDog `ce5b0f06d27cc1bbfb8357f6ecb06682649b8e644f45911e9ae2c793321e19f4`;
  DPD `cb257d55e4d71429f112aefce05d50346c4e4966c997805b1bab28804129eb5c`.
  These archives are now registered in `resources/incidents/SOURCES.tsv`.

## Confirmed findings to act on

1. **Chapter 00 factual error:** the opener says Tree of Alpha edited the source-account field
   in the 0.0243 ETH/BTC test. His thread says he changed `product_id` from ETH-EUR to BTC-USD
   and *left* the ETH source and EUR target account IDs unchanged. Coinbase's retrospective
   separately illustrates the bug by editing a source account. Do not combine these two
   demonstrations into one sequence. The exact 0.0243 figures and lack of BTC are supported by
   the researcher's thread (tweets 2–4); add claim rows and correct the opener. The sentence
   asserting that the order form and matching engine assumed other checks is inference, not in
   the public record; narrow or label it.
2. **Chapter 01 source overreach:** the Peloton report shows editable workout IDs and personal
   data from private profiles, but its full field list covers several API issues. Do not assign
   every listed field to `POST /stats/workouts/details` without endpoint-specific evidence.
   The prose also says IDs appeared on every leaderboard; the source says IDs could be found in
   the mobile app. The current test is a fixed list of routes, so adding `/v3` would *not* make
   the test fail automatically. The chapter claims Go's compiler prevents direct store calls,
   but `Store.Invoice` is exported in the current single-package service.
3. **Chapter 01 test gap:** its exercise includes refund quote/confirm and client-selected
   tenant-list cases that the current pilot service and `service/ch01_test.go` do not implement.
   Do not claim every exercise case is executable yet. Decoy/list and invoice/PDF cases do run.
   A later implementation decision must either add the missing service paths and both-mode tests
   or revise the exercise; do not write tests that fail merely because those routes are absent.
4. **Chapter 02 source and code gap:** the BrewDog primary report supports the shared bearer
   token, customer-ID substitution, field exposure, version timeline, and researcher account of
   no customer notification. Archive and claim-gate these facts. A copied app key proves only
   possession of the key, *not* that a request came from a real build of the app. The chapter's
   `service/ch02_test.go` claim is premature: no Chapter 2 service or tests exist. Its D/E
   tenant-key pair currently produces the same invoice result in both cases and needs a distinct
   observable outcome to satisfy GUIDANCE L8. `/v1` is still serving until Chapter 9, so its
   exercise wording should not imply retirement now.
5. **Chapter 03 source/class and code gap:** the DPD researcher report directly supports the
   unauthenticated map endpoint, derivable postcode, session token, and underlying JSON with
   recipient contact details. The named full-name/email/mobile field list comes from a secondary
   report, not the archived primary text, unless the primary screenshot is checked and archived.
   The primary article itself says both “within one week” and “within three weeks”; prefer the
   dated disclosure timeline and avoid an unqualified duration. The incident's main exploit
   includes a weak postcode gate; property exposure is present but whether it alone carries API3
   deserves an explicit editorial decision. No Chapter 3 service or tests exist, so code
   placeholders and the claimed `service/ch03_test.go` are provisional. The draft's £1,800/£640
   amounts conflict with the current pilot fixture amounts (7300/9100); resolve the fixed table
   before code is built.

## Next actions, in order

1. Add the three source-manifest rows, then add `checks/claims/02.tsv` and `03.tsv` with primary
   source locators. Add Chapter 00 rows for the researcher's thread.
2. Create the dated review report required by `IMPROVE-PROMPT.md`, writing each chapter's
   ranked findings *before* its edits. Correct Chapter 00's mixed sequence and commit it.
3. Make supported sentence-level Chapter 01 corrections. Run `go test ./...` from `service/` and
   `./verify.sh`; record the unresolved exercise implementation gap, then commit Chapter 01.
4. Tighten Chapters 02 and 03 only where sources and existing design support it. Keep their Go
   placeholders; do not claim unbuilt tests run. Record design issues needing the author's call
   and commit each chapter's review separately, as the prompt requests.
5. Leave Chapter 04 untouched in this pass. Do not push.

Primary public records: [Coinbase retrospective](https://www.coinbase.com/blog/retrospective-recent-coinbase-bug-bounty-award),
[Tree of Alpha's thread](https://threadreaderapp.com/thread/1495014902582362112.html),
[BrewDog disclosure](https://www.pentestpartners.com/security-blog/free-brewdog-beer-with-a-side-order-of-shareholder-pii/),
[DPD disclosure](https://www.pentestpartners.com/security-blog/dpd-package-sniffing/).
