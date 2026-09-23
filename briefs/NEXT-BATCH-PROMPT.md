# Prompt to the main session after context compaction: draft chapters 5, 6, 7

Paste this to Claude in a fresh or compacted session, working directory ~/book23.

---

Book 23 ("Security Rebook", ~/book23, branch main) has chapters 0–4 drafted and a Codex
review-and-improve pass may have run on 0–3 (check `git log` for `review: chNN` commits and
`review/codex-*.md`). Draft the next three chapters — **5, 6 and 7** — with three fresh
general-purpose Fable subagents in parallel, one per chapter. Do not fork; give each agent only
the files below.

Before launching, read `briefs/DRAFTING-BRIEF.md` yourself and, if chapters 2–4 or CONTEXT.md
changed since it was written (compare `git log -- CONTEXT.md briefs/ chapters/`), update its
"State of Ledger" table so it reflects every repair now in force. Also add a row for chapter 4's
Instagram switch if CONTEXT's register still says X/Twitter.

Each agent's prompt, verbatim apart from the chapter-specific block:

> You are drafting one chapter of Book 23 in /home/diablo/book23. Read, in order:
> `briefs/DRAFTING-BRIEF.md`, `CONTEXT.md`, `GUIDANCE.md` §2, `chapters/01-whos-asking.md` (voice
> and form reference only), your chapter's row in `PLAN.md` §3, and `briefs/02.md`–`04.md` for the
> state of the service. Fetch and read the ORIGINAL disclosure with WebSearch/WebFetch; Domoney's
> *Defending APIs* (text at ~/book11/workspace/defending-apis/source.txt) is only the pointer.
> Write one sentence giving the incident's exact mechanism from the record and why it fits the
> assigned OWASP class. If the record cannot carry the class, STOP the chapter and write only the
> brief with sourced alternatives; never invent incident detail. Otherwise write
> `chapters/NN-slug.md` (1,600–2,100 words; `{{excerpt:chNN-name}}` placeholders for every Go
> block; claims comment at the top with source URLs; one incident, one class, the bug on Ledger,
> why the local fix is insufficient, a second path that reuses an earlier repair inside the worked
> example, an exercise after `<!--mission-->` with a plausible decoy and a near-identical pair with
> opposite outcomes and worked answers tracing input → check → response, one-line credit) and
> `briefs/NN.md` (mechanism and fit; Ledger objects reused; handlers and mode behaviour; exercise-
> case matrix with vulnerable and fixed outcomes; excerpt names; proposed CONTEXT additions with no
> look-alike numbers; claims to gate; anything unverified). Write only those two files. Do not
> commit. Report in under 200 words: mechanism sentence, word count, open issues.

Chapter-specific blocks:

- **Chapter 5 — "Same Door, Different Verb" (API5, function-level authorization).** Incident:
  the 2022 campus access-control system where endpoints taking student IDs reached admin
  functions (Domoney case 2; find the researcher's own write-up and the TechCrunch report; verify
  the vendor/product name from the record before using it). Ledger: `DELETE /v2/invoices/{id}`
  and the tenant-admin routes share the GET handler's check from chapter 1; same URL, different
  verb; the `/admin` prefix that `/v1` never had. Repair: authorization middleware with an
  explicit role or "public" declaration on every route, so a grep proves nothing is defaulted.
  Dana (Cedar admin) versus Alice is the natural pair.
- **Chapter 6 — "Every Request Was Valid" (API6, sensitive business flows).** Incident is
  **open**. Spend the first part of your budget on a source hunt: a primary account (researcher
  write-up, vendor disclosure, court filing, bug-bounty report) of excessive automated use of an
  authorised flow — ticket or product purchasing bots, sign-up or referral abuse, mass account
  creation, coupon or refund flows — where each request was valid and the abuse was the volume
  or sequence. Not a race condition (the Starbucks gift-card case was rejected for that reason)
  and not a rate-limit brute force (that is chapter 4). If you find one that the record supports,
  write the chapter: Ledger's refund quote `q-771` → confirm flow repeated beyond the business
  limit; the UI limits the flow, the API does not; repair: flow-level limits and abuse controls,
  idempotency only for duplicate confirms. If you find none, write `briefs/06.md` with the
  candidates you examined and why each failed, and stop.
- **Chapter 7 — "The Server That Fetched for You" (API7, SSRF).** Incident: Shopify Exchange
  screenshot-service SSRF, HackerOne 2018, $25,000 — read the disclosed report; its surface (a
  screenshot service) differs from Ledger's, say so. Ledger: the tenant webhook "test" endpoint
  fetches a caller-controlled URL (`https://hooks.cedar.example/ledger` is Cedar's fixed value);
  redirects and resolved addresses cross the boundary after the initial URL check. Repair: one
  outbound policy at one egress (public addresses only, no redirects followed), demonstrated with
  a fake resolver and transport. Not Capital One (Book 17's).

When all three return: read each draft and brief; if chapter 6 returned only a brief, record the
open state in CONTEXT's register; commit the new files together on main (`git add chapters/05*
chapters/06* chapters/07* briefs/05.md briefs/06.md briefs/07.md`), message "Prose: exploratory
chapters 5–7 with implementation briefs". Then extend `briefs/DRAFTING-BRIEF.md`'s state table
with rows 5–7 and commit. Do not push. Report to the author: mechanism sentences, word counts,
incident switches, open issues, and whether chapter 9 (the second pilot, from the ACMA filing in
`resources/incidents/09/` and `checks/claims/09.tsv`) has been written yet — if not, that is the
main session's next task, not an agent's.
