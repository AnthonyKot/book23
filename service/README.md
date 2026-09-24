# Ledger pilot fixture

Ledger is a deliberately small, deterministic teaching service. `NewApp` exposes the Chapter 1
before/after state, including the client-selected invoice list and refund quote/confirm sequence.
Quote and refund amounts are integer pence. `NewChapter9App` is still a pilot that starts after
the Chapter 1 loader repair; it does not yet contain Chapters 2–8. Authentication uses opaque
fixture tokens; all data is in memory; tests make no network requests.

This fixture proves only the response and inventory outcomes encoded here. It cannot establish
that a real gateway matches its repository configuration, that traffic telemetry is complete,
that an undiscovered host does not exist, that a production identity system authenticated a
caller correctly, or that a real service was exploited. Production retirement needs external
probes, gateway and DNS changes, telemetry over an agreed observation window, and an owner who
accepts the residual risk. The fixture tokens and host filter are not production security
controls.
