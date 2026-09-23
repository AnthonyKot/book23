# Resume prompt for the Book 23 review

Work in `/home/diablo/book23`. Read `review/CHECKPOINT-2026-09-23.md` and
`review/IMPROVE-PROMPT-04-07.md`, then run `git status --short` and `git log --oneline -8` before
making changes. Resume the active Chapters 04–07 review; the Chapter 00–03 pass is complete and
must not be repeated. Check the checkpoint's active-pass section for which chapter was last
committed and which sources were archived.

For each remaining chapter in order, archive the primary records named in its header, register
SHA-256 checksums in `resources/incidents/SOURCES.tsv`, create `checks/claims/NN.tsv` with source
locators, write ranked findings in `review/codex-2026-09-23-04-07.md` before prose/brief fixes,
then make only the sentence-level changes authorized by the prompt. Record whether the exercise
pair gives visible different outcomes, whether the brief is implementable without guessing, and
whether it matches the State of Ledger table. Commit each chapter with `review: chNN`; update the
checkpoint after each chapter. Service code exists only for Chapters 01 and 09; do not pretend
the Chapter 04–07 cases run or build their service in this pass.

Run `cd service && go test ./...`, `./verify.sh` and `git diff --check`. Do not edit chapters 00–03
or 08–10, rebuild the site, push or publish. Finish with the single most important unresolved
issue per chapter and links to the report and checkpoint.
