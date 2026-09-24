# Review checkpoint — 2026-09-24

Active prompt: `review/IMPROVE-PROMPT-08-10.md`; scope Chapters 08, 09 and 10 only. The
working tree was clean at `810c563` and matched `origin/main` when this pass began. Chapters
00–07 are reviewed; their reports are `review/codex-2026-09-23.md` and
`review/codex-2026-09-23-04-07.md`. Service code exists for Chapters 01 and 09 only.

Read the prompt, CONTEXT, PLAN, GUIDANCE, the Chapter 1 incident choice and handover, the
drafting brief, prior review reports, and the file register. No new source has been fetched or
chapter changed yet. Chapter 08's old prefix-guard dependency must be checked against the
revised Chapter 05 brief; Chapter 09 has no `briefs/09.md`, so its existing service and tests
are the implementation contract; Chapter 10 needs Kiln and SwissBorg primary archives.

Next: read Chapters 00–10 in numeric order with their available briefs and existing tests, then
work 08 → 10. For each chapter, source-gate the incident, write ranked findings before edits,
make only authorized sentence-level changes, run `cd service && go test ./...`, `./verify.sh`,
and `git diff --check`, update this checkpoint, and commit with `review: chNN`. Do not build new
service code, rebuild the site, push or publish in this pass.

Chapter 08 completed: UpGuard's original Power Apps report is archived and hashed, with ten
located claims. The chapter now describes the Microsoft default change without a rollout date,
and exercise C gives the fixed request an opaque 404 while a separate settings assertion rejects
`Debug:true`. The brief uses CONTEXT's `CORSOrigins`, specifies a debug-only internal reason
helper, and removes the stale Chapter 5 prefix guard. The health routes and web-app origin are
registered in CONTEXT without changing previous values. `go test ./...`, `./verify.sh`, and
`git diff --check` passed; Chapter 8's Go service remains unbuilt. Next: Chapter 09; check all
four printed excerpts against marked service blocks, every exercise case against both-mode tests,
and claims 09-10 through 09-14 against the archived court PDF.

Chapter 09 completed: the five later claim rows were checked against the court PDF's scanned
concise statement; 09-11/09-14 were clarified and 09-15 gates the May 2024 proceeding date.
All three printed Go blocks match their marked excerpts exactly. The prose now keeps the Target
Domain's 2017 exposure separate from the joint vulnerable state by June 2020, distinguishes
address/identity-document subsets from fields accessed for all affected customers, and preserves
Chapter 4's shared-handler lookup budget. `service/ch09_test.go` now checks invoice data in 200
responses and its absence in 404 responses for every exercise case in both modes. `go test ./...`,
`./verify.sh`, and `git diff --check` passed. Next: Chapter 10; archive Kiln and SwissBorg's own
accounts, gate its claims, then check the stored FX rate and refund quote against Chapter 6.
