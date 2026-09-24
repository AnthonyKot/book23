# Ledger pilot fixture

Ledger is a deliberately small, deterministic teaching service. `NewApp` exposes the Chapter 1
before/after state, including the client-selected invoice list and refund quote/confirm sequence.
Quote and refund amounts are integer pence. `NewChapter2App` keeps the Chapter 1 repair in both
modes and demonstrates session-derived identity, a shared mobile key, and tenant integration keys.
Its login route uses hard-coded fixture passwords and its sessions are in memory; it is a teaching
fixture, not deployable authentication. `NewChapter3App` keeps the earlier repairs and demonstrates
stored-row exposure versus role-specific response views, typed PATCH requests, and a raw `Invoice`
JSON guard. The Chapter 1 and 2 stages retain their original four-field response shape.
`NewChapter4App` keeps the first three repairs and demonstrates OTP challenge locks, separate
IP throttling, a 50-row page cap, and 30/min email-lookup budgets at both gateway and shared
handler. Its in-memory limits, fixture OTP code, and simulated text delivery are teaching devices.
`NewChapter5App` composes the earlier repairs and declares access for every registered route;
its vulnerable mode skips the tenant-admin role decision while retaining tenant scope.
`NewChapter6App` retains that role decision in both modes. Its fixed confirm enforces a tenant's
daily refund allowance in the in-memory store and parks over-limit quotes for admin approval;
the vulnerable confirm omits that allowance. Reminder sends are counted as fixture events only.
Tests make no network requests.
`NewChapter7App` adds a shared outbound fetcher for webhook tests and the `/v1` PDF logo path.
Fixed mode resolves once, refuses non-public addresses, pins the approved dial address, ignores
environment proxies and stops at redirects. Chapter 7 tests inject DNS, dial and HTTP responses;
the intentionally vulnerable mode uses default redirect behavior.
`NewChapter8App` retains the earlier repairs and adds explicit production and staging
settings, a route-table public allow-list, an admin health probe, and a debug error-body
counterexample. Chapter 8 keeps the public fixture login route in its allow-list alongside
health and the two OTP routes, so the earlier session lesson remains operable.
`NewChapter9App` composes Chapters 1–8 in both modes. Its vulnerable mode still registers
`/v1` and forwards it on the unlisted staging host; fixed mode removes those routes and
rejects that host. The runtime inventory includes every registered route, while the three
printed Chapter 9 code blocks remain the chapter's focused exercise examples.
`NewChapter10App` begins from Chapter 9's fixed route and host policy in both modes.
`PollRates` models one hourly poll with injected time and outbound
transport in tests. Vulnerable mode accepts the partner's arbitrary rate map and converts
refunds at the latest rate; fixed mode checks a typed EUR row, retains the last good rate,
and quotes refunds at the invoice's recorded rate. No scheduler or real partner is included.

This fixture proves only the response and inventory outcomes encoded here. It cannot establish
that a real gateway matches its repository configuration, that traffic telemetry is complete,
that an undiscovered host does not exist, that a production identity system authenticated a
caller correctly, or that a real service was exploited. Production retirement needs external
probes, gateway and DNS changes, telemetry over an agreed observation window, and an owner who
accepts the residual risk. The fixture tokens and host filter are not production security
controls.
