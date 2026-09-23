# Review-and-improve prompt for the author to run in Codex CLI, from ~/book23

Run after `prose` has been merged into `main` (chapters/, briefs/, review/ present alongside
service/, checks/, resources/):

    codex exec -m gpt-5.6-sol -s workspace-write -C ~/book23 "$(cat review/IMPROVE-PROMPT.md)" < /dev/null

---

You are reviewing and improving Book 23, "Security Rebook", in this directory. **Scope of this run: chapters 00, 01, 02 and 03 only** (chapters/00-*.md … 03-*.md). Service code exists only for chapters 1 and 9; chapters 2 and 3 keep their `{{excerpt:...}}` placeholders and are reviewed as prose plus brief. Do not build service code in this run. Read first:
CONTEXT.md, PLAN.md, GUIDANCE.md, docs/CH01-INCIDENT-CHOICE.md, docs/HANDOVER.md,
briefs/DRAFTING-BRIEF.md. Then every chapter in chapters/ in numeric order, its brief in briefs/,
the tests in service/*_test.go, the claims in checks/claims/, and the archived sources in
resources/incidents/.

Work one chapter at a time, in order. For each chapter, first write findings, then apply the
fixes you are allowed to apply, then commit that chapter's changes with a message that begins
"review: chNN". Never push.

## What to check (GUIDANCE.md §2 is the test; these are its instances)

1. Source fidelity: every incident fact (date, number, endpoint, quote, timeline, consequence)
   must be found in resources/incidents/NN/ and have a row in checks/claims/NN.tsv with a
   locator. Facts not found are UNSUPPORTED; sentences that go beyond the record are OVERSTATED.
2. Class fit: state the mechanism from the source in one sentence and whether it supports the
   chapter's OWASP class.
3. Spine: delete "Ledger" and the cast mentally; does the worked example collapse, or is the
   service only a label? Which earlier object or repair does the chapter reuse inside its example?
4. Mechanism: the missing check is shown, the reason it is structurally skipped is given, and a
   second path or step repeats the skip. Quote the hinge sentence.
5. Exercise: list the cases; name the plausible decoy and the near-identical pair with opposite
   outcomes; every verdict traces input → check (or its absence) → response or state change.
6. Code and tests: every printed Go block matches a marked excerpt in service/ exactly; no
   `{{excerpt:...}}` placeholder remains in a chapter whose service code exists; every exercise
   table row has a test asserting both the vulnerable and the fixed outcome. Run
   `cd service && go test ./...` and record the result.
7. Cold read, in persona: a developer who can follow a Go handler and an HTTP request but has not
   learned to trace authorization across routes. Where did attention drop, could you predict the
   decoy, can you state the missing check in one sentence, what is the next door?
8. Continuity: does the chapter add a new problem or restate the previous one with new nouns?
   Any name or number that conflicts with CONTEXT.md's fixed table?

## What you may change

- Facts: correct any UNSUPPORTED or OVERSTATED sentence to what the archived source supports, or
  delete it. Add missing rows to the claims file with locators. Never add an incident detail that
  is not in an archived source; if the record is silent, the prose is silent.
- Code and tests: fill placeholders from the marked excerpts; fix drift between a printed block
  and its excerpt by changing the chapter, not the code; add missing test cases so every table
  row is asserted in both modes; fix a test whose assertion disagrees with the table only after
  deciding, and recording, which one is right.
- Exercise: if the decoy or the lookalike pair is missing, add one case in the chapter's own
  style, with its worked answer and its test row.
- Prose: sentence-level only. Fix an untraced verdict by adding the trace; fix an asserted hinge
  by adding the one or two sentences that show it; cut a hedge; fix a wrong term. Keep the
  author's voice, headings, order and examples. Do not restructure a chapter, replace its
  incident, rewrite its opening, or change its length by more than about 10%. If a chapter needs
  more than that, say so in the findings and leave it.
- CONTEXT.md: add fixed values the briefs propose (they say "add to CONTEXT") if no two values
  look alike; record incident switches (ch. 4 → Instagram 2019) in the register and decision
  record; never change an existing value.

## What you must not do

Do not rewrite prose beyond sentence level. Do not add paragraphs of explanation, summaries,
"key takeaways", or a second incident. Do not touch chapters whose brief says they are stopped
or whose incident is still open. Do not edit GUIDANCE.md or PLAN.md except the ch. 4 register
row. Do not push.

## Output

For each chapter, append to review/codex-$(date +%F).md: the findings (severity BLOCK / FIX /
NOTE, ranked), what you changed (file and line), what you left for the author with the reason,
and a one-paragraph verdict: ready for the author's read or not, and the single most important
reason. End the file with a list of every proposed change you did not make because it exceeded
the sentence-level rule, so the author can decide.
