# Nobody Counted

<!-- Incident switched from the register's X/Twitter 2022 candidate to Instagram 2019 (Laxman Muthiyah): see briefs/04.md for the category-fit reasoning. Claims to gate in checks/claims/04.tsv: six-digit recovery code, 10-minute validity; endpoint /api/v1/accounts/account_recovery_code_verify/; a burst of ~1,000 attempts from one address had ~250 accepted before the rest were refused (~200 per IP); the limit was keyed per source address; Facebook paid a $30,000 bounty and fixed it quickly; disclosed July 2019. Source: Laxman Muthiyah, "How I Could Have Hacked Any Instagram Account", thezerohack.com; corroborated by Threatpost and WeLiveSecurity, July 2019. Aside: HackerOne report 1439026 (zhirinovskiy, submitted 1 Jan 2022, disclosed 11 Feb 2022, $5,040 bounty), which notes that a caller could turn one lookup into a database of the user base; 5.4M records offered for sale July 2022 (The Record, 22 Jul 2022). -->

In 2019 Laxman Muthiyah looked at how Instagram let a person back into an account they had been
locked out of. You gave a phone number, Instagram texted a six-digit code, and you had ten minutes
to enter it. Six digits is a million codes. Instagram knew that and had put a limit on the endpoint
that checks the code: a burst of about a thousand tries from one address saw only around 250
accepted before the rest were refused. Two hundred or so tries from one place, against a million
codes, decides nothing.

The weakness Muthiyah reported was not in the code, the ten-minute window, or the size of the
number. It was in what the limit counted. It counted tries *per source address*, and a source
address is not a scarce thing: cloud providers rent thousands by the hour. The limit answered
"how many times has this address guessed?" when the question that protects the account is "how many
times has anyone guessed *this code*, for *this account*, in these ten minutes?" Muthiyah showed
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
  challenge, the challenge is locked and a new code must be requested. Ten thousand addresses share
  one counter, because the counter hangs off the account, not the connection.
- **A used challenge is spent.** A correct code, or a lock, retires the challenge. Without that, an
  attacker who rents a new address also gets a fresh five tries.
- **The per-address limit stays.** It is still worth having against noisy clients and cost blowups.
  It is simply not what keeps the account safe. Two limits, two jobs.

## If it's just a counter, why is it a whole category?

Because a service has many scarce things, and each one needs its own counter, keyed to itself.

- **The limit lands on the loud route, not the enumerating one.** Everyone rate-limits `login`.
  Almost nobody rate-limits the lookup that quietly confirms whether an email belongs to a user —
  which is exactly the endpoint that turned into a five-million-record list at Twitter (below).
- **A limit per IP is not a limit per target.** The Instagram limiter was real; it counted the
  wrong subject. This is the mistake that hides in a passing load test.
- **Reads cost too.** A page size nobody caps is a resource bug: `GET /v2/invoices?limit=1000000`
  makes the database assemble a million rows for one request. Ledger caps `limit` at 50.
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

Twitter shows the loud-route mistake at scale. A researcher reported in January 2022 (HackerOne
1439026, a $5,040 bounty) that submitting a phone number or email to a duplicate-account check in
the Android login flow returned the account's ID, even for users who had turned discoverability
off. One such lookup is a minor leak. The reporter's own impact note is the API4 point: with basic
scripting, a caller could enumerate a large part of the user base into a phone-and-email-to-account
table. In July 2022, 5.4 million such records were offered for sale. The login page was surely
rate-limited. The lookup underneath it was the route that mattered, and it was not.

<!--mission-->
## Exercise: which counter protects the thing?

Use Ledger as built so far. The OTP challenge locks after 5 wrong codes per account; `limit` caps
at 50; lookup routes get 30 requests/minute. Here are four routes, written as route → limiter key.

```text
A. POST /v2/auth/otp/verify   -> limit 5 per (account_id, challenge_id)
B. GET  /v2/invoices?email=X  -> limit 30/min per client IP
C. GET  /v1/invoices?email=X  -> (no limiter registered)
D. GET  /v2/invoices?limit=N  -> cap N at 50; no per-caller budget
```

For each: name the scarce thing the route spends, say what the limiter counts, and mark it
adequate or not. Then find the pair that look almost identical and don't behave the same.

**Check your answer.**

- **A is adequate.** The scarce thing is guesses against one account's active code. The key is the
  account and the challenge, so renting new addresses buys nothing; the fifth wrong code locks it.
- **B is the near-identical twin of A that gets it wrong.** The scarce thing is *lookups that
  confirm an email belongs to a customer*, which is a per-target concern. Keying the limit on
  client IP counts the wrong subject: a caller rotating addresses enumerates freely, exactly the
  Twitter shape. Adequate against a noisy single client, not against enumeration. Fix: also cap
  distinct email lookups per authenticated caller, and prefer answers that don't confirm existence.
- **C is not adequate; it's the forgotten door.** Same data as B, no limiter at all, because the
  gateway's route table never listed v1. Register it, or route v1 through the same shared budget as
  v2. This is Chapter 1's `/v1` again — unwatched routes escape whatever you add.
- **D is the plausible decoy.** Capping `limit` at 50 looks like the same fix as B, and it is worth
  having: it stops one request from assembling a million rows. But it is a different scarce thing
  (work per request, not lookups per caller), and it does nothing about a caller who sends the
  50-row request thousands of times. A cap on size is not a budget on frequency.

If you marked B adequate because "it has a rate limit," reread the Instagram flow. Instagram had a
rate limit too. It counted the caller when it needed to count the code.

*Incident from Laxman Muthiyah, "How I Could Have Hacked Any Instagram Account" (thezerohack.com,
2019); Twitter aside from HackerOne report 1439026 and The Record, July 2022. Method after Colin
Domoney, Defending APIs (Packt, 2024).*
