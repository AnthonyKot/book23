<!--
Claims to gate in checks/claims/10.tsv.
Sources: Kiln, "Re-enablement of Kiln services and security incident information", 7 Oct 2025,
https://www.kiln.fi/post/re-enablement-of-kiln-services-and-security-incident-information (vendor
post-mortem); SwissBorg, "Kiln Breach (Sep 2025): Security Impact on SwissBorg", 17 Nov 2025,
https://swissborg.com/blog/swissborg-security-update-kiln-breach (consumer's account); joint
statement, 8 Sep 2025, https://swissborg.com/blog/joint-statement-kiln-x-swissborg-regarding-sol-incident;
Kiln announcement, 8 Sep 2025, https://www.kiln.fi/post/sol-incident-swissborg---announcement.
- Entry point: a GitHub access token of a Kiln infrastructure engineer; the actor created and immediately
  deleted branches to trigger CI/CD workflows and harvested stored secrets and cloud credentials (Kiln).
- The actor injected a payload into a running Kubernetes pod hosting the Kiln Connect API, modifying the
  logic of one API endpoint (Kiln).
- The endpoint returned a malicious transaction in addition to the expected "deactivate stake" transaction;
  it changed the withdrawal authority of the Solana stakes only if the existing withdrawal authority of the
  stake account in the POST call held balances above 150k SOL (Kiln).
- One customer used Kiln Dashboard to unstake SOL on 31 Aug 2025; the transaction was forwarded from the
  Dashboard to their custody solution and approved by a quorum of signatories (Kiln).
- Incident detected 8 Sep 2025; the initial statement said it may have involved unauthorized access
  to a staking wallet; Solana funds were improperly removed; SwissBorg paused staking (joint statement).
- Over 192,000 SOL unstaked in the fraudulent transaction (SwissBorg). No dollar figure: neither
  primary account gives one.
- Kiln "consistently recommended" customers decode transactions to verify integrity before signing and
  provides a decoding tool (Kiln); SwissBorg chose not to rely on it because it was not integrated in the
  dashboard, not open-sourced or verifiable, and did not meet basic security standards (SwissBorg).
- SwissBorg: the actor did not breach SwissBorg's wallet infrastructure; the incident occurred entirely
  within Kiln's systems (SwissBorg).
- No evidence of any other malicious transaction or other customer affected (Kiln).
- Ledger's rate feed is invented for the analogy and is never presented as Kiln's or SwissBorg's system.
-->

# What You Swallowed

On 31 August 2025 SwissBorg did a routine thing. They used the
dashboard of Kiln, the company that ran their Solana staking, to unstake some
SOL. Behind the dashboard sat Kiln's Connect API: you POST the stake account and what you want
done, and it returns a transaction for you to sign. The dashboard forwarded the transaction to
SwissBorg's custody system, a quorum of their signers approved it, and it went on-chain.

The transaction was not what they had asked for. Some time before, an attacker had obtained a GitHub
access token belonging to a Kiln infrastructure engineer. With it, by Kiln's own account, they
created and immediately deleted branches to trigger CI workflows, harvested the secrets and cloud
credentials those workflows held, and used those to inject a payload into the running pod that
served the Connect API. The payload changed the logic of one endpoint. It still returned the
"deactivate stake" transaction a caller expected. Alongside it, it returned a second transaction
that reassigned the *withdrawal authority* of the caller's stake accounts to an address the
attacker controlled, and only when the existing withdrawal authority for the stake account in
the POST held stake balances above 150,000 SOL. Other requests could return exactly what the
caller expected. SwissBorg got that, and one thing more.

On 8 September a fraudulent transaction triggered the unstaking of over 192,000 SOL, by
SwissBorg's account. Kiln and SwissBorg published a joint
statement the same day; Kiln's post-mortem followed in October and SwissBorg's own account in
November. Two lines from those accounts are the whole lesson. Kiln: it had "consistently
recommended" that customers decode transactions to verify their integrity before signing, and
offered a tool for it. SwissBorg: it had chosen not to rely on that tool, because it was not
integrated into the dashboard, not open-source or verifiable; and, separately, that the attacker
"did not breach SwissBorg's wallet infrastructure" and the incident "occurred entirely within
Kiln's systems".

The two accounts differ on whether the offered decoder was usable, but together they show the
consumer's boundary. SwissBorg says its systems were not breached. Its custody system took what
the partner's API returned and acted on it. OWASP calls this **unsafe consumption of APIs**,
API10: a service gives data from a
third-party API more trust than it would give a request body, so a partner that is compromised,
buggy, or simply different from what you assumed writes straight into your state. The request you
validated went out. The response you did not validate came back, and you signed it.

## Where the bug lives: a row from the partner, written into the total

Ledger bills in pounds, but Cedar has customers who pay in euros, so an invoice carries a
`Currency` field. To record a euro invoice in pence, Ledger
needs a rate, and it gets one from a partner feed at `rates.partner.example`, which answers
`GET /latest` with a row like `{"EUR": 0.92}`: pounds per euro. A poller fetches the row every hour
and the store keeps it. Here is the poller and the invoice creation that uses it, before this
chapter's fix.

{{excerpt:ch10-rates-vulnerable}}

Read it the way you now read a request handler. The fetch is a bare `http.Get`, the call the SSRF
chapter banned from handlers; the poller predates `egress` and nobody moved it. The body is
decoded into a `map[string]float64` and the whole map becomes the store's rate table, every key,
whatever it is. Then `CreateInvoice` looks up `rates[inv.Currency]` and, if a rate exists for the
invoice's currency, multiplies. For Cedar's invoice 412, €300.00, the row `{"EUR": 0.92}` gives
£276.00 and 27600 pence go into the row. Nothing checks that 0.92 is a plausible number of pounds
per euro. Nothing checks that the only key is `EUR`. Nothing keeps the rate: the invoice stores
27600 and forgets where the number came from.

So the partner's response is the input, and four responses show four ways to swallow it. A row of
`{"EUR": 92}` bills the customer £27,600.00. A row of `{"EUR": -0.92}` stores a negative total,
which leaves the refund flow with a negative remaining balance. A row of
`{"EUR": 0.92, "GBP": 0.5}` carries the
rate you asked for, correct to the penny, and beside it a key you never asked for, and
`CreateInvoice` applies it: from that hour every new Cedar invoice in pounds is halved, because "if
a rate exists for the currency" was the only condition. And a `302` sends the poller to whatever
address the `Location` header names, from which a row comes back with the same shape and no way to
tell. The third case is Kiln's in miniature: the expected thing arrived, the extra thing arrived
beside it, and the code applied everything that arrived.

The fix treats the feed like a request body from a stranger:

{{excerpt:ch10-rate-row}}

{{excerpt:ch10-rates-fixed}}

Three details decide whether this is real.

- **The response is decoded into a declared type, with unknown fields rejected.** `rateRow` has one
  field, `EUR`. `DisallowUnknownFields` is the call the property chapter used on `InvoicePatch`;
  here it is pointed the other way, at what comes in from a partner rather than a client. A row
  with a `GBP` key is refused whole, not trimmed.
- **The value is bounded by a range Ledger declares, not by what the partner sent last time.**
  The fixed poller accepts a euro rate only inside `0.70` to `1.10`, Ledger's assumed bounds for
  this example. That refuses the decimal slip and negative value here; a real feed needs a range
  its owner can revise as conditions change. Outside it, the poller keeps the last good rate and
  pages Ops. It does not bill at the new one.
- **The invoice records the rate it used.** `FXRate` and the original `AmountEUR` are stored next
  to the pence. A bad rate that gets through anyway is then a query, not an archaeology. This is
  a record Ledger can inspect before a later quote uses the rate.

## If it is one range check, why is it a whole class?

Because the range check guards one value on one path, and the trust that caused the bug is not
about a value. It is the assumption that a partner's response is already what you meant.

- **The partner's shape is not your schema.** `{"EUR": 0.92}` is what the feed usually sends, not
  a contract, and a map that accepts anything makes every future key a silent feature. The only
  schema that protects you is the one you declared.
- **A correct partner can still be the wrong input.** Kiln's endpoint was compromised, but the same
  consumer code would have swallowed a Kiln bug, a schema change, or a provider's decimal-point
  mistake. The check is for *your* invariants, whatever the partner's reason for violating them.
- **The partner's verification tool is not your check.** Kiln offered a decoder; SwissBorg declined
  it as unsuitable for its workflow. The parties disagree on that tool; Ledger's response
  validation belongs in code its own team can inspect.
- **A response is an outbound request's other half.** The SSRF chapter fixed where Ledger's
  requests go and said nothing about what comes back. `egress` refuses the `302`, and that is the
  right first line, but a partner that answers `200` with a hostile body is beyond anything a
  destination policy can see.

## The second time the rate is read

The rate feed is Ledger's fourth outbound call, the one the SSRF chapter said would arrive "next
quarter" and either use the one client or stand out in review. In the fixed build it uses `egress`,
so the resolve, the private-address check, and the no-redirect rule are inherited. That closes the
`302` case without a line of URL code in the poller, the way the PDF logo fetch was closed.

The path that catches people is the refund. The quote from the business-flows chapter,
`POST /v2/refunds/quote` with `{invoice_id, amount}`, loads the invoice with `LoadInvoiceFor` and
checks the amount against what remains in pence. For a euro invoice, it must first convert the
requested euro amount to pence. Here is how the vulnerable build did it:

{{excerpt:ch10-refund-quote-vulnerable}}

It converts at `store.rates["EUR"]`: today's rate, whatever the poller last stored. Fix the poller
and this is still wrong, in two ways. A bad rate still in the store can reach a quote; even with a
perfect feed, a full €300.00 refund on an invoice billed at 0.92 becomes £285.00 at 0.95. The
remaining-balance check then rejects it against the £276.00 invoice, although the customer asked
for the full amount originally billed in euros. The repair is the one the refund chapter made for
the quote, applied to the rate: confirm uses the invoice stored with the quote; quote uses the rate
stored with the invoice.

{{excerpt:ch10-refund-quote-fixed}}

The quote stores that converted pence amount, so `ConfirmRefund` and its tenant allowance never
convert it again. An older euro invoice without a recorded `FXRate` needs reconciliation before
a quote can be issued. For new invoices, `store.rates` is consulted in one place: at creation.
That is the shape of `LoadInvoiceFor` and `egress`: one door, one check, and every other path
reads the checked result rather than the raw source.
Recording what passed the boundary, and reading only the record afterward, is what makes the
boundary hold for the paths you have not thought of yet.

<!--mission-->
## Exercise: which of these changes what Cedar's customers are billed?

Ledger's poller fetches `GET https://rates.partner.example/latest` every hour and, in the
vulnerable build, stores whatever map comes back; invoices are created at the stored rate.
Each case starts from the last good rate of 0.92 and creates fresh fixture versions of Cedar's
invoice 412 (€300.00) and invoice 104 (£1,800.00) after the fetch. A teammate proposes this fix:
"reject any rate greater than 10." Below are five partner responses. For each: say what the poller
stores, what invoices 412 and 104 are recorded as, and mark whether the teammate's cap stops the
damage and whether the chapter's
fixed build does. You do not need a running service.

```text
1. 200  {"EUR": 92}
2. 200  {"EUR": 0.92}
3. 200  {"EUR": -0.92}
4. 200  {"EUR": 0.92, "GBP": 0.5}
5. 302  Location: https://rates.partner.example/v2/latest
        (the redirected page answers 200 {"EUR": 0.95})
```

**Check your answer.**

- **1 is the decoy.** It is the response the cap was written for: 92 is greater than 10, so the
  teammate's rule drops it, and the fixed build drops it too, because 92 is outside `0.70`–`1.10`.
  Vulnerable build: row `{"EUR": 92}` → no check → `rates["EUR"] = 92` → invoice 412 stored as
  2760000 pence, £27,600.00; 104 untouched. It is conspicuous on the first invoice.
- **2 is the baseline, and must keep working.** Row → decoded into `rateRow{EUR: 0.92}`, no
  unknown fields → inside the band → stored with `AsOf` → invoice 412 recorded as 27600 pence with
  `FXRate 0.92`; 104 at 180000 pence. Both builds and the cap produce this. A fix that breaks it has
  broken invoicing.
- **3 is vulnerable, and the cap misses it.** −0.92 is not greater than 10. Vulnerable build:
  `rates["EUR"] = -0.92` → invoice 412 stored as −27600 pence, and the refund flow's
  remaining-balance check sees a negative amount. Fixed build: −0.92 is below `0.70` → row
  refused, last good rate kept, Ops paged; 412 is created at 0.92. The cap guarded the end of the
  range someone had imagined.
- **4 and 2 are the near-identical pair.** The euro rate is byte-for-byte the same, and the cap
  passes 4: every value is under 10. Vulnerable build: the whole map is stored, `CreateInvoice`
  finds a rate for `GBP`, and invoice 104, a pound invoice that should never be converted, is
  stored as 90000 pence, £900.00; 412 is correct. Fixed build: `DisallowUnknownFields` rejects the
  row at `GBP` before any value is read; the last row stands and both invoices are correct. The
  discriminator is not a number. It is whether the code accepted a field it never declared, which
  is why no check on the *value* of `EUR` can reach it. This is Kiln's shape: the thing you asked
  for was correct, and the thing beside it was the attack.
- **5 is closed by the earlier chapter, not this one.** Vulnerable build: `http.Get` follows the
  redirect and stores 0.95 from a location the poller never chose; invoice 412 becomes 28500
  pence. The cap never sees a redirect; it checks values, not fetches. Fixed build: `egress`
  returns the `302` as a result and does not follow it; the last good 0.92 stays, so invoice 412
  is 27600 pence. If you marked 5 as this chapter's bug, notice that its fix lives in a different
  function from the fix for 3 and 4: where the request goes is egress policy; what the response
  may say is this chapter.

If you marked 4 safe because the euro rate was right, that is the transaction SwissBorg's signers
approved: the deactivate they had asked for was in it.

*Incident from Kiln, "Re-enablement of Kiln services and security incident information" (7 Oct
2025), and SwissBorg, "Kiln Breach (Sep 2025): Security Impact on SwissBorg" (17 Nov 2025). The
method follows Colin Domoney, Defending APIs (Packt, 2024).*
