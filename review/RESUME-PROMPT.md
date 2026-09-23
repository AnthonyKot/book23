# Resume prompt for the Book 23 review

Work in `/home/diablo/book23`. Resume the paused review of Chapters 1–3 using
`review/IMPROVE-PROMPT.md`, with its Chapter 00 source correction as the first prerequisite.
Read `review/CHECKPOINT-2026-09-23.md` before anything else. Do not restart the source hunt:
the Tree of Alpha, BrewDog, and DPD records were already fetched into
`resources/incidents/00/`, `02/`, and `03/`. Check `git status` and recent commits first so
you do not repeat work that may have happened since the checkpoint.

Apply the review prompt one chapter at a time, writing findings before fixes and making a
separate `review: chNN` commit after each completed chapter. First resolve Chapter 00's mixed
Coinbase/researcher sequence and claim rows. Then review Chapters 1, 2, and 3 in order:
source fidelity, class fit, Ledger continuity, the decoy and contrasting pair, exact Go
excerpts where code exists, and both-mode tests. Add the new archive rows to
`resources/incidents/SOURCES.tsv` and create claims ledgers for Chapters 2 and 3.

Respect the implementation boundary: service code currently exists only for Chapters 1 and 9.
Chapter 2 and 3 excerpts and test tables are proposals; do not claim they run. The Chapter 1
refund and client-filter exercise paths also lack service tests. Record a precise remaining
implementation task if completing it would exceed this review's scope. Run `go test ./...`
from `service/`, `./verify.sh`, and `git diff --check` before reporting.

Preserve the author's voice and the named Ledger spine. Make evidence-backed changes now;
leave larger incident or architecture choices as explicit BLOCK findings with options for the
author. Do not touch Chapter 4, push, or publish. Finish with links to the changed files and
the single most important unresolved issue for each chapter.
