# Nobody Counted

<!-- Incident claims are gated in checks/claims/04.tsv against Muthiyah's Instagram disclosure and its original-publication metadata. The researcher observed both a race hazard and IP rotation; his report does not expose the exact limiter implementation. The Twitter aside is gated against HackerOne #1439026 and X's later incident update; neither source establishes a missing lookup rate limit. -->

In 2019 Laxman Muthiyah looked at how Instagram let a person back into an account they had been
locked out of. You gave a phone number, Instagram texted a six-digit code, and you had ten minutes
to enter it. Six digits is a million codes. Instagram knew that and had put a limit on the endpoint
that checks the code: in one test, about a thousand tries saw around 250
accepted before the rest were refused. Two hundred or so tries from one place, against a million
codes, decides nothing.

The weakness Muthiyah reported was not in the code, the ten-minute window, or the size of the
number. It was in what the limit failed to count across requests. He used both concurrent requests
and rotating source addresses; an address is not a scarce thing. The observed limit answered too
little about "how many times has anyone guessed *this code*, for *this account*, in these ten
minutes?" Muthiyah showed
Facebook a proof of concept that the number of guesses an attacker could afford, spread across
enough addresses, was on the same order as the number of codes. Facebook paid a $30,000 bounty and
fixed it quickly.

That is OWASP's **API4, unrestricted resource consumption**: an endpoint lets a caller spend a
scarce resource — here, guesses against a short secret — faster and more often than the business
can afford, because nothing counts the spending against the thing that matters.

## Where the bug lives: a limiter that counts the wrong thing

Ledger has the same short-secret flow. When a customer forgets their password, Ledger texts a
six-digit one-time code and checks it at `POST /v2/auth/otp/verify`. Here is that handler before
this chapter's fix.

{{excerpt:ch04-otp-vulnerable}}

There is a limiter in front of it. Read what it keys on.

{{excerpt:ch04-limiter-per-ip}}

It counts requests per client address. Ben, verifying his own login from his phone, is held to a
sane rate, which is what the limiter was added for. But the count resets with every new address,
and the code it is guarding is only a million wide. The limiter is busy and the account is not
protected, because the two are measured against different things.

The fix counts against the account and the code, not the caller:

{{excerpt:ch04-otp-fixed}}

Three details matter.

- **The budget is per target, not per source.** After five wrong codes for one account's active
  challenge, the challenge is locked for the rest of its ten-minute window; requesting a new code
  cannot reset that budget early. Ten thousand addresses share one counter for that target.
- **A used challenge is spent.** A correct code, or a lock, retires the challenge. Without that, an
  attacker who rents a new address also gets a fresh five tries.
- **The per-address limit stays.** It is still worth having against noisy clients and cost blowups.
  It is simply not what keeps the account safe. Two limits, two jobs.

## If it's just a counter, why is it a whole category?

Because a service has many scarce things, and each one needs its own counter, keyed to itself.

- **The limit lands on the loud route, not the enumerating one.** Limiting `login` would not by
  itself bound a separate lookup that confirms whether an email belongs to a user. Twitter's
  duplicate-account check illustrates the privacy cost of such a lookup (below).
- **A limit per IP is not a limit per target.** Instagram's observed limit was real; concurrent
  requests from rotating IPs got around it. This can hide in a passing load test.
- **Reads cost too.** A page size nobody caps is a resource bug: `GET /v2/invoices?limit=1000000`
  could demand huge work as the store grows. Ledger caps `limit` at 50.
- **Forgotten routes have no budget at all.** The `/v1` invoice routes from Chapter 1 are behind
  `LoadInvoiceFor` now, so they can't leak across tenants — but no per-route budget was ever wired
  to them. A route the current team doesn't watch is a route nobody is counting.

## The door you forgot: the same limiter, two keys

Here is where it bites on Ledger. After the OTP fix, the team adds the 30-requests-per-minute
budget to the lookup routes, and wires it through the gateway Ops runs. They test it on
`GET /v2/invoices?email=…`, the customer-facing lookup, and it holds: the 31st request in a minute
gets a 429.

Three weeks later, support automation starts timing out. The cause is `GET /v1/invoices?email=…` —
the v1 lookup, still serving the old mobile app, never added to the gateway's route table. It
reaches the same data with no budget on it. The gateway can only limit routes it knows about, and
v1 was Chapter 1's forgotten door in a new disguise: last time it skipped the *authorization*
check; this time it skips the *counting*.

So the rule for a rate limit is the rule for an authorization check, one chapter on. Finding every
door is step one either way. A budget, like `LoadInvoiceFor`, has to sit where every route passes
— at the gateway *and* in the shared handler — not on the one route a reporter happened to name.

## An aside on the enumerating route

Twitter shows what repeated lookups could expose at scale. A researcher reported in January 2022 (HackerOne
1439026, a $5,040 bounty) that submitting a phone number or email to a duplicate-account check in
the Android login flow returned the account's ID, even for users who had turned discoverability
off. One such lookup is a minor leak. The reporter's own impact note is the API4 point: with basic
scripting, a caller could enumerate a large part of the user base into a phone-and-email-to-account
table. X later confirmed that this bug had been exploited and said the affected set reported in
2022 contained 5.4 million accounts. The public report does not establish the lookup's rate limit.

<!--mission-->
## Exercise: which counter protects the thing?

Use Ledger at the point where Ops has limited the v2 lookup but has not yet added v1 to the
gateway. The OTP challenge locks after 5 wrong codes per account, without a new request resetting
the ten-minute window; `limit` caps at 50. Here are four routes, written as route → limiter key.

```text
A. POST /v2/auth/otp/verify   -> limit 5 per (account_id, challenge_id)
B. GET  /v2/invoices?email=X  -> limit 30/min per client IP
C. GET  /v1/invoices?email=X  -> (no limiter registered)
D. GET  /v2/invoices?limit=N  -> cap N at 50; no per-caller budget
```

For each: name the scarce thing the route spends, say what the limiter counts, and mark it
adequate or not. For B and C, send 31 requests from one IP in a minute, then rotate IPs; which
response changes? That is the near-identical pair.

**Check your answer.**

- **A is adequate.** The scarce thing is guesses against one account's active code. The key is the
  account and the challenge, so renting new addresses buys nothing; the fifth wrong code locks it.
- **B is half of the B/C pair.** The scarce thing is lookups that confirm whether an email belongs
  to a customer. From one IP, request 31 gets a 429; rotating addresses resets the IP counter, so
  the same caller can keep asking. Fix: also budget lookups by the authenticated tenant and route,
  and prefer answers that do not confirm existence.
- **C is not adequate; it's the forgotten door.** Same data as B, no limiter at all, because the
  gateway's route table never listed v1. From the same IP, request 31 is still served: same lookup,
  opposite outcome. Register v1 with the tenant-and-route budget too. This is Chapter 1's `/v1`
  again — unwatched routes escape whatever you add.
- **D is the plausible decoy.** Capping `limit` at 50 looks like the same fix as B, and it is worth
  having: it stops one request from demanding an unbounded page. But it is a different scarce thing
  (work per request, not lookups per caller), and it does nothing about a caller who sends the
  50-row request thousands of times. A cap on size is not a budget on frequency.

In the completed fix, both B and C have the 30/min tenant-and-route budget, so changing IP no
longer resets either counter. If you marked B adequate just because it had a rate limit, reread
the Instagram flow: the observed limit still allowed many guesses across addresses.

*Incident from Laxman Muthiyah, "How I Could Have Hacked Any Instagram Account" (thezerohack.com,
2019); Twitter aside from HackerOne report 1439026 and The Record, July 2022. Method after Colin
Domoney, Defending APIs (Packt, 2024).*
