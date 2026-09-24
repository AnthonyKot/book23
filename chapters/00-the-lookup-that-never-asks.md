# The Lookup That Never Asks

<!-- claims to gate in checks/claims/00.tsv: February 2022; Tree of Alpha; ETH-EUR order from the UI, request carries product, source and target account ids; changed product_id to BTC-USD and left both account ids; "0.0243 ETH to sell 0.0243 BTC"; fills matched, live order book; later test with source account changed to a SHIB account, 50 BTC limit sell; retrospective quote "missing logic validation check ... mismatched source account"; validation checked balance, not asset; patch validated the same day (11:42 AM to 4:01 PM); $250,000 largest to date; no malicious exploitation found. Sources: Tree of Alpha's thread (Thread Reader archive); Coinbase retrospective blog. -->

In February 2022 a trader who posts as Tree of Alpha was trying Coinbase's new advanced trading
interface. He placed an ETH-EUR order from the web page and looked at the request his browser sent.
It carried three things: a product, the pair being traded, and a source and a target account. He
changed the product to BTC-USD and left both accounts as they were, ether as the source, euros as
the target, expecting an error, because his account was not allowed to trade that pair. The order
went through. In his words, he had used 0.0243 ETH to sell 0.0243 BTC on a pair he did not have
access to, without holding any BTC, and the fills on the live order book matched. Before reporting
it he tried the same mismatch from the other side: he moved some SHIB into an account, named it as the
source, and placed a limit order to sell 50 BTC.

Coinbase's own retrospective names the cause with unusual precision: a missing logic validation
check in one API endpoint "allowed a user to submit trades to a specific order book using a
mismatched source account". Its validation checked that the named source account had the balance
the order needed. It did not check that the account held the asset the order book trades. The
report arrived on 11 February; a patch was validated and released that afternoon. The bounty was
$250,000, then the largest Coinbase had paid, and the company said it found no malicious use.

Notice what the bug was not. It was not a break-in. The trader was logged in to his own account,
spending his own money, sending a request the server was built to accept. The documented balance
check passed. The damage came from the check that nobody wrote: it looked at the account it was
handed, and nothing asked whether that account and that order book were talking about the same
asset.

That is the question this book asks of every request, in one form or another:

**Which check did this request assume had already happened?**

Sometimes the missing check is "does this record belong to the caller?". Sometimes it is "is this
the same object the previous step approved?", or "who counted how many times this was called?", or
"does anyone know this route still exists?". The OWASP API Security Top 10 gives those gaps names,
and the chapters follow its order. The names are less important than the habit: find the input the
server trusted, find the check it skipped, and then find the second route or second step where the
same check is skipped again, because there nearly always is one.

To make that habit concrete, the chapters share one small system. **Ledger** is an invoicing API run
by an invented company. Two customer tenants, **Cedar** and **Birch**, use it. **Alice** works for
Cedar, **Ben** for Birch, and **Dana** is Cedar's administrator. Invoice 104 belongs to Cedar and
invoice 205 to Birch. Ledger has a current API under `/v2`, an older one under `/v1` that a mobile
app still calls, and a small platform team, Ops, that runs the gateway in front of both. It is
written in Go with nothing but the standard library, so every check is visible in the handler, and
every handler you read in this book exists as a file whose tests run.

Ledger is a teaching example, not a reconstruction of anyone's system. This is not a reference
on tools or process either: no scanners, no programme, no checklist beyond the one question. The incidents that open
each chapter are real and are told from the original public record. What Ledger does is let you see
the same mistake in fifteen lines, fix it, and then go looking for the door you forgot.
