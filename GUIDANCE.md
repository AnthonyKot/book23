# Guidance — what Books 1, 2 and 11 taught, applied to Book 23

Status: written 2026-09-23 alongside PLAN.md, before any chapter exists. PLAN.md says what the
book is; this file says how to write it so the mistakes of the earlier books are not repeated.
Precedence: CONTEXT.md (once it exists) → PLAN.md → this file → chapter briefs.

## 1. Lessons, each with the evidence it came from

**L1. A book is connected in its bodies, not at its seams.** Book 2 had prerequisite links and a
"why the next chapter follows" paragraph on every chapter and still read as fifteen stitched
lessons: after ch. 3 every chapter opened its own fresh puzzle and the text almost never leaned
on an earlier object (from ch. 10 on, one in-body back reference per chapter). Its later spine
revision reuses fixed objects and numbers (the pair from ch. 4 is *the* pair in ch. 7, 8, 9, 15;
ch. 14 corrects ch. 13's measured noise), but that change is not independent reader evidence.
Rule for Book 23: **the service is the spine**. A
chapter must reuse at least one named object from an earlier chapter *inside its worked example*,
not only in a bridge sentence. "Lab A's buttons" style relabelling does not count; a number or
function the reader already computed does.

**L2. A spine can be costume.** In Book 2's rework, ch. 1–3, 6 and 10–12 gained "Lab A's" in
front of nouns and nothing else; ch. 4→5, 4→7→8→9→15 and 13→14→15 were real. Test for each
chapter: delete every mention of the service's name; if the calculation still stands unchanged,
the spine is decoration there. Either make the chapter depend on the service's state or say
honestly that this class of bug is service-independent and keep the chapter short.

**L3. Keep the physics when adding the story.** The same rework silently dropped Book 1's
spin-magnet and polariser examples and ch. 8's polariser note, and turned ch. 10's idea-bridge
into a qubit count. Whenever a worked example is re-set on the service, diff the old and new
text for lost mechanism: the "three details that matter", the reason the bug is structural, the
one-sentence hinge. Story is added to explanation, never swapped for it.

**L4. Two numbers that look alike will be confused.** Book 2 ended up with a pair visibility of
0.80 and a fibre arrival probability of 0.80 in the same chapter. Book 23's fixed numbers (invoice
IDs, tenant names, rate limits, token lifetimes, bounty amounts) go in one table in CONTEXT.md and
no two of them may coincide or differ only by a suffix. Invoice 104 and 205 already obey this.

**L5. Compression is the failure mode of a careful writer.** Book 2's first three-reviewer read
(2026-09-11) found every number correct and every chapter "partly clear, 3/5": the hinge was
asserted in one sentence, symbols appeared before definition, half the "worked answers" stated
the result without the intermediate line. The fix was un-compressing about fifteen load-bearing
steps, not adding chapters. For Book 23: every exercise answer shows the request, the check that
runs (or doesn't), and the response; every "vulnerable" verdict names where the ID comes from
and what compares it to the caller. A verdict without that trace is a stated, not worked, answer.

**L6. The reader wants one problem per essay.** Book 1 was rebuilt from four-author comparisons
into one-question essays and the author's verdict on the pilots was "very readable"; Book 20's
diagnosis was "chapters are concatenation of smaller pieces". One incident, one bug class, one
twist, one exercise per chapter. If a second class of bug shows up in the incident (Coinbase was
BOLA *and* a multi-step trust failure), it is a beat of the same story, not a second section.

**L7. A model's "I would keep reading" is not reader evidence.** The 2026-09-23 Codex review of
Book 2 wrote as a Book 1 reader and then admitted it was not one. Panel reviews check correctness,
compression and lost mechanism (L3, L5); they do not certify the reading experience. **The author
reads the two pilots before any further chapter is drafted** (Book 1's gate; it worked).

**L8. The exercise is the chapter's proof.** The Book 11 essay's exercise had five handlers, two
decoys (B looks unsafe and is safe; E looks like B and is not), and a closing trap sentence. That
is the useful pattern: the reader must distinguish the check from a lookalike that omits it.
Book 23 exercises may use handlers, request sequences, gateway rules, or partner responses as
the mechanism requires. Include a plausible decoy, and a near-identical pair with opposite
outcomes where it reveals the hinge; do not force five handlers or the same final sentence into
every chapter. The Book 11 critique (2026-09-22) asked for a safe-looking decoy handler.

**L9. Verification is the design.** Book 2's `check_calculations.py` recomputes every printed
number; Book 13 runs its labs against real toolchains. Book 23's equivalent: the service is real
code and executable exercise outcomes are tests. Each case asserts what the vulnerable mode
actually exposes and what the fixed mode prevents; both assertions pass only when the contrast
is real. An arbitrary nonzero test exit is never counted as evidence of a vulnerability. A
chapter whose code is not in `service/` is not finished.

**L10. Sources: pointer vs. evidence.** Book 11's essays carry no citations in the body and one
credit line; Book 17 fetches primary disclosures and quote-gates every number. Book 23 combines
them: prose reads clean, but `checks/claims/NN.tsv` maps each incident number to a fetched
primary source in `resources/incidents/NN/`. Domoney's retellings are the pointer; the
researcher's write-up or the vendor's disclosure is the evidence. Also check incident-to-class
fit: Coinbase's $250,000 report was a mismatch between a trade's source account asset and order
book, not cross-user BOLA; Starbucks' gift-card race is not itself API6's excessive access; the
named SiriusXM and Hyundai findings do not establish one API10 partner chain. Never invent a
detail of a real incident; if the record is silent, the prose is silent.

**L11. Lengths hold when set in advance.** Book 2's ±15% rule kept the rework from ballooning
(largest growth +14%). Chapter target 1,600–2,100 words; the pilots calibrate it. Keep later
chapters near that range, but never cut the decisive request, comparison or state change just to
hit a word count.

**L12. Lanes.** Mechanical and well-specified work (incident fetch, pitches, applying a panel
report, exercise-table-to-test conversion) goes to codex sol; judgment-heavy writing (the chapter
prose, the twist) is the main session, one chapter at a time. Codex's Book 2 spine run proves the
first; the review fixes prove the second was still needed. Long runs save as they go: one WIP
commit per chapter, never push from an agent.

## 2. The chapter test (run before a chapter is called done)

1. Delete the service's name everywhere: does the worked example collapse? (L2)
2. Diff against the incident's primary source: does its mechanism fit the OWASP class, and is
   every number and consequence supported? (L10)
3. Read the exercise answers: does each trace the untrusted input or event, the check, and the
   response or state change? (L5)
4. Is there a plausible decoy and a meaningful contrasting pair where the mechanism allows? (L8)
5. Does the chapter reuse a named earlier object in its worked example? (L1)
6. Does `verify.sh` pass while proving both the vulnerable outcome and fixed outcome? (L9)
7. Is the chapter near the pilot's length without compressing its hinge? (L11)
8. Read the old and new text of any re-set example side by side: mechanism lost? (L3)

## 3. Files this guidance expects

```
book23/
  PLAN.md            what the book is, chapter register, decisions needed
  GUIDANCE.md        this file
  SEED-PROMPT.md     bounded instructions for the two pilots
  CONTEXT.md         (next) decision record, cast + fixed-number table, correction log
  service/           the Go billing service, vulnerable/ and fixed/ per chapter
  service/chNN_test.go   the chapter's exercise table as httptest cases
  checks/claims/     NN.tsv, one row per incident number or quote
  resources/incidents/NN/   fetched primary sources (gitignored if licensed)
  chapters/          static HTML, one per chapter
  verify.sh          links, structure, and explicit vulnerable/fixed assertions
```
