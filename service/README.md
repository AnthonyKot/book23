# Ledger pilot fixture

Ledger is a deliberately small, deterministic teaching service. `NewApp` exposes the Chapter 1
before/after state, including the client-selected invoice list and refund quote/confirm sequence.
Quote and refund amounts are integer pence. `NewChapter2App` keeps the Chapter 1 repair in both
modes and demonstrates session-derived identity, a shared mobile key, and tenant integration keys.
Its login route uses hard-coded fixture passwords and its sessions are in memory; it is a teaching
fixture, not deployable authentication. `NewChapter3App` keeps the earlier repairs and demonstrates
stored-row exposure versus role-specific response views, typed PATCH requests, and a raw `Invoice`
JSON guard. Historical stages retain their original four-field response shape. `NewChapter9App`
is still a pilot that starts after the Chapter 1 loader repair; it does not yet contain Chapters
2–8. `NewChapter4App` keeps the first three repairs and demonstrates OTP challenge locks, separate
IP throttling, a 50-row page cap, and 30/min email-lookup budgets at both gateway and shared
handler. Its in-memory limits, fixture OTP code, and simulated text delivery are teaching devices.
`NewChapter5App` composes the earlier repairs and declares access for every registered route;
its vulnerable mode skips the tenant-admin role decision while retaining tenant scope. Tests make
no network requests.

This fixture proves only the response and inventory outcomes encoded here. It cannot establish
that a real gateway matches its repository configuration, that traffic telemetry is complete,
that an undiscovered host does not exist, that a production identity system authenticated a
caller correctly, or that a real service was exploited. Production retirement needs external
probes, gateway and DNS changes, telemetry over an agreed observation window, and an owner who
accepts the residual risk. The fixture tokens and host filter are not production security
controls.
