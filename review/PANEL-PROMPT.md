# Panel prompt — Codex CLI, run from ~/book23 after `prose` is merged into `main`

Run:

    codex exec -m gpt-5.6-sol -s workspace-write -C ~/book23 "$(cat review/PANEL-PROMPT.md)" < /dev/null

(`workspace-write` so it can write its report; it must not edit chapters.)

---

You are an independent reviewer of Book 23, "Security Rebook", in this directory. You did not
write its prose and you will not rewrite it. Your output is a findings report, not an edit.

Read first: CONTEXT.md, PLAN.md, GUIDANCE.md, docs/CH01-INCIDENT-CHOICE.md, docs/HANDOVER.md.
Then read every file in chapters/ in numeric order, the briefs in briefs/, the tests in
service/*_test.go, the claims files in checks/claims/, and the archived sources in
resources/incidents/.

Write your report to review/codex-$(date +%F).md and nothing else. Do not modify any chapter,
brief, test, or planning file. If you find a defect you could fix in one line, describe it; do not
apply it.

Review every chapter present against the eight-point chapter test in GUIDANCE.md §2, in this
order, and report per chapter:

1. **Source fidelity.** For every fact about the real incident (date, number, endpoint, quote,
   consequence, timeline), find it in resources/incidents/NN/ or checks/claims/NN.tsv. Report
   each fact you cannot find as UNSUPPORTED with the sentence quoted. Report any sentence that
   goes beyond the record as OVERSTATED. Check the claims file itself: rows without a locator
   are findings.
2. **Class fit.** In one sentence, state the incident's mechanism from the archived source and
   whether it supports the chapter's OWASP class. If it does not, say what class it does support.
3. **Spine test.** Delete the service's name and its cast mentally: does the worked example
   collapse, or does the chapter merely say "Ledger's" in front of generic nouns? Name the earlier
   object or repair the chapter reuses inside its worked example, if any.
4. **Mechanism.** Does the chapter show the missing check, why it is structurally skipped, and a
   second path or step where the same check is skipped again? Quote the hinge sentence. If the
   hinge is asserted rather than shown, say where.
5. **Exercise.** List the cases. Identify the plausible decoy and the near-identical pair with
   opposite outcomes; if either is missing, say so. For each verdict, check it traces input →
   check (or its absence) → response or state change. A verdict that states the outcome without
   the trace is a finding.
6. **Code and tests.** Every printed Go block must match a marked excerpt in service/ exactly;
   report any drift, and any `{{excerpt:...}}` placeholder still present. Every row of the
   chapter's exercise table must have a test case asserting both the vulnerable and the fixed
   outcome; report rows without one, and test cases whose assertion does not match the table.
   Run `cd service && go test ./...` and report the result verbatim.
7. **Cold read.** Read the chapter once more in the persona of a developer who can follow a Go
   handler and an HTTP request but has not learned to trace authorization across routes. Report:
   the first paragraph where your attention dropped and why; whether you could predict the
   decoy's verdict before the answer; whether you can state the missing check in one sentence
   afterwards; and what the next door to inspect would be. Answer as that reader, not as an
   auditor.
8. **Continuity.** For chapters after the first: does it add a new problem, or restate the last
   chapter's lesson with new nouns? Does it use a specific object or repair from an earlier
   chapter? Any conflict with CONTEXT.md's fixed-number table (a name or number that differs)?

Severity for each finding: BLOCK (unsupported fact, wrong class, placeholder in a chapter
presented as done, test/table mismatch), FIX (mechanism asserted not shown, missing decoy or
pair, untraced verdict, spine as costume), NOTE (style, length, wording). Rank BLOCK first.

Be adversarial toward the chapter's own claims and toward GUIDANCE.md's rules where a rule
produces a worse chapter; say so explicitly if you think a rule should change. Do not praise.
Do not suggest rewrites longer than one sentence. End with a one-paragraph verdict per chapter:
ready for the author's read, or not, and the single most important reason.
