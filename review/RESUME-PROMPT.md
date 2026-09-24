# Resume prompt for the Book 23 review

Work in `/home/diablo/book23`. Read `review/CHECKPOINT-2026-09-24.md` and
`review/IMPROVE-PROMPT-08-10.md`, then run `git status -sb` and `git log --oneline -8`.
Resume the active review at the next uncommitted chapter in the checkpoint. Chapters 00–07
are complete and out of scope.

Work one chapter at a time in order. Archive the primary records named in Chapters 08 and 10,
register hashes in `resources/incidents/SOURCES.tsv`, and create located claims ledgers before
judging source fidelity. Chapter 09 already has archives and claims; check rows 09-10 through
09-14 against the court PDF. Write findings in `review/codex-2026-09-24-08-10.md` before prose
or brief fixes. Test visible exercise contrasts, implementability, and State of Ledger fit.
Chapter 09's marked excerpts and both-mode Go tests must be checked byte for byte and row by
row. Commit each chapter with `review: chNN`, updating the checkpoint after each.

Run `cd service && go test ./...`, `./verify.sh`, and `git diff --check`. Do not build service
code for Chapters 08 or 10, edit chapters outside scope, rebuild the site, push, or publish.
Finish with the main unresolved issue per chapter and links to the report and checkpoint.
