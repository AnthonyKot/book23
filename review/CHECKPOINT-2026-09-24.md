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
