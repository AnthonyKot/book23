# Pilot handoff

## Lane A → Lane B

Lane A stopped before the opener and chapter prose. The source gate, Ledger fixture, paired tests,
and pre-prose verification scaffold are ready for the writing worktree.

### Source gate and mechanism sentences

The archived-source manifest is `resources/incidents/SOURCES.tsv`; claim-level evidence is in
`checks/claims/00.tsv`, `checks/claims/01.tsv`, and `checks/claims/09.tsv`.

1. **Coinbase:** Coinbase’s retrospective says its Retail Brokerage endpoint checked the named
   source account’s balance but did not check that the account’s asset matched the requested order
   book; its example gives both asset accounts to one user, so this is an account/asset consistency
   failure, not cross-user BOLA.
2. **Peloton:** Jan Masters showed that IDs in `POST /stats/workouts/details` could be taken from the
   mobile app, manually changed, and sent; making the endpoint require login still left the data
   available to other authenticated Peloton members, including data from profiles marked private.
3. **USPS Informed Visibility:** Brian Krebs independently confirmed that a logged-in USPS user
   could change search parameters, including wildcards, to query other users’ account details; this
   is a strong search-shaped object-authorization backup, although the original researcher was
   anonymous and the account does not establish that all roughly 60 million records were taken.
4. **Tinder:** Max Veytsman showed that the user-by-ID response returned a high-precision
   `distance_mi` value and that queries from spoofed locations could locate a user; the decisive
   failure is excessive location precision, not evidence that the caller lacked permission to view
   a potential match’s object.

**Gate result:** Peloton still earns Chapter 1. It supplies the exact editable ID request, the
cross-member result after authentication, and the partial login-only repair. USPS remains the
sourced backup. Tinder does not carry this class.

The source pages were archived on 2026-09-23. Coinbase and the ACMA landing page required rendered
snapshots because their sites blocked or stalled a direct archival request; the manifest says so
and records checksums. Peloton, USPS, and Tinder are direct HTML responses. The Optus mechanism is
from the redacted concise statement annexed to the Federal Court’s 19 June 2024 order in
VID429/2024, not from a press retelling.

### Optus evidence for Chapter 9

The filed concise statement alleges the following sequence. Its paragraph 8 identifies three
Target APIs intended to return information after customer authentication. Paragraph 9 says the
Target Domain was dormant and unused since 2017 but not decommissioned until after the attack.
Paragraphs 10–11 allege that one coding error affected the Main and Target domains, and that Optus
fixed the Main Domain in August 2021 without detecting or fixing the same issue on the Target
Domain. Paragraph 13 says the Target Domain sat dormant and vulnerable for two years. Paragraph 14
says the attacker bypassed access controls and sent requests to the Target APIs. Paragraph 16 says
the domain was decommissioned on 21 September 2022. Every Chapter 9 sentence must present these as
ACMA’s filed allegations, not adjudicated facts, and must leave the filing’s redactions unfilled.

Archive: `resources/incidents/09/optus-filed-order-and-concise-statement.pdf`. Claim locators and
short evidence fragments are in `checks/claims/09.tsv`.

### Exact Ledger paths and states

- `service/go.mod` — Go 1.22 module, standard library only.
- `service/model.go` — Alice/Cedar, Ben/Birch, Dana/Cedar admin; invoices 104/Cedar and 205/Birch;
  `Store.LoadInvoiceFor`.
- `service/app.go` — authentication fixture, `/v2`, still-serving Chapter 1 `/v1` read and PDF
  routes, `NewApp(Vulnerable)`, `NewApp(Fixed)`, and the cumulative `NewChapter9App` states.
- `service/inventory.go` — declared-host, code-route, gateway, traffic, deprecation, and unlisted
  `ledger-staging.internal` fixtures plus reconciliation.
- `service/ch01_test.go` — every invoice-serving route crossed with Alice→104, Alice→205, and
  Ben→104, plus the server-filtered list decoy and Dana fixture.
- `service/ch09_test.go` — host/route outcomes, inventory findings, and fixture-to-router checks.
- `service/README.md` — boundaries of what the toy can and cannot prove about a deployment.
- `verify.sh` — pilot test-group, claim/archive checksum, excerpt-marker, and deferred prose
  structure/link checks.

`NewApp(Fixed)` is the Chapter 1 state: it protects `/v2`, `/v1/invoices/{id}`, and
`/v1/invoices/{id}/pdf` with the same loader while `/v1` still serves authorized callers.
`NewChapter9App(Vulnerable)` starts from that repair, so Alice already gets 404 for Birch invoice
205 on `/v2`; it still serves `/v1` through the public and staging surfaces. Only
`NewChapter9App(Fixed)` retires `/v1` and rejects the staging host while retaining
`LoadInvoiceFor` on `/v2`.

All Go blocks currently intended for quotation are bounded by unique
`// excerpt: <chNN-name>` / `// end excerpt` markers. Lane B should extract, not retype, them. If
prose needs a different block, add markers around the exact source before quoting it.

### Exercise-case matrix

Chapter 1:

| Case | Vulnerable | Fixed | Purpose |
|---|---|---|---|
| Alice → `/v2/invoices/104` | 200 Cedar | 200 Cedar | Near-identical allowed half |
| Alice → `/v2/invoices/205` | 200 Birch | 404, no Birch data | Near-identical denied half; missing caller/object check |
| Ben → `/v2/invoices/104` | 200 Cedar | 404, no Cedar data | Reverses the two-user attempt |
| Same three callers/objects through `/v1/invoices/{id}` | Same vulnerable leak/own success | Own 200; cross-tenant 404 | Proves `/v1` was protected, not retired |
| Same three callers/objects through `/v1/invoices/{id}/pdf` | Same leak in PDF | Own 200; cross-tenant 404 | Second forgotten door |
| Alice → `/v2/me/invoices` | Only invoice 104 | Only invoice 104 | Plausible decoy: direct store query is safe because the server derives the tenant filter from Alice |
| Dana → `/v2/invoices/104` | 200 Cedar | 200 Cedar | Cedar tenant-admin fixture |

Chapter 9:

| Case | Before Chapter 9 repair | After Chapter 9 repair | Purpose |
|---|---|---|---|
| Public host, `/v2/invoices/104` | 200 | 200 | Current-route decoy remains live |
| Public host, `/v2/invoices/205` as Alice | 404 | 404 | Chapter 1 loader survives both states |
| Public host, `/v1/invoices/104` | 200 | 404 | Deprecated is serving versus retired |
| `ledger-staging.internal`, `/v1/invoices/104` | 200 | 404 | Unlisted host discovered and removed |
| `ledger-staging.internal`, `/v2/invoices/104` | 404 | 404 | Near-identical opposite in the before state: staging forwarded `/v1`, not every version |
| `ledger-preview.internal`, `/v1/invoices/104` | 404 | 404 | Plausible unknown-host decoy |
| Reconcile code + gateway + traffic | Three deprecated surfaces and one unlisted host | No active mismatch | State proof independent of one HTTP probe |

The matrices deliberately distinguish Chapter 1’s authorization question from Chapter 9’s
inventory/retirement question. The invented Ledger behavior is an analogy, never a reconstruction
of Peloton or Optus.

### Passing command

From `service/`:

```text
go test ./...
```

passes (`ok ledger`). From the repository root, `./verify.sh` runs verbose tests and requires the
named Chapter 1 and Chapter 9 groups to pass. Until Lane B adds HTML, it reports the chapter/link
checks as pending; once any chapter HTML exists, it requires one `00*.html`, one `01*.html`, and one
`09*.html`, plus title structure, both exercise markers, and valid local/external links.

### Unresolved claims and limits

- The Optus case is a regulator’s pleaded account. Preserve “ACMA alleges” until a later court
  source establishes a finding or admission. The public filing redacts system details.
- Peloton’s text establishes cross-member exposure after login and shows private-profile response
  examples. Do not infer database ownership fields, server code, modification of records, or that
  every listed field came from this one endpoint.
- The USPS number describes accounts potentially exposed. The record does not say all were
  downloaded, and the underlying researcher is anonymous.
- Tinder’s endpoint returned potential-match objects intentionally; its evidence supports a
  location-property/privacy lesson, not Chapter 1’s missing per-object authorization check.
- Coinbase’s example is one user’s two asset accounts. Keep it out of the cross-user BOLA claim,
  and prefer its explicit timestamps or “a matter of hours” over “within six hours.”
- The Ledger fixture cannot discover an unknown real host, prove production gateway parity or
  telemetry completeness, validate a real identity system, or prove exploitation. The full limit
  statement and required production evidence are in `service/README.md`.
