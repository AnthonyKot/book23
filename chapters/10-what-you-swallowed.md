<!-- Incident claims: checks/claims/10.tsv -->

# What You Swallowed

On 31 August 2025 SwissBorg, a Swiss crypto wealth platform, did something routine. They used the
dashboard of Kiln, the company that ran their Solana staking, to unstake some SOL. Behind the dashboard sat Kiln's Connect API: you POST the stake account and what you want
done, and it returns a transaction for you to sign. The dashboard forwarded the transaction to
SwissBorg's custody system, a quorum of their signers approved it, and it went on-chain.

The transaction was not what they had asked for. Some time before, an attacker had obtained a GitHub
access token belonging to a Kiln infrastructure engineer. With it, by Kiln's own account, they
created and immediately deleted branches to trigger CI workflows, harvested the secrets and cloud
credentials those workflows held, and used those to inject a payload into the running pod that
served the Connect API. The payload changed the logic of one endpoint. It still returned the
"deactivate stake" transaction a caller expected. Alongside it, it returned a second transaction
that reassigned the *withdrawal authority* of the caller's stake accounts to an address the
attacker controlled, and only when the authority behind the stake account in the POST held more
than 150,000 SOL in stake. Below that line a caller got exactly what they asked for. SwissBorg,
by its own account the first of Kiln's clients over the line to sign an unstake, got that and one
thing more.

On 8 September the authority that had changed hands was used. SwissBorg says it detected, within minutes,
a fraudulent transaction that unstaked more than 192,000 SOL. Kiln and SwissBorg published a
joint statement the same day; Kiln's post-mortem followed in October and SwissBorg's own account in
November. Two lines from those accounts are the whole lesson. Kiln: it had "consistently
recommended" that customers decode transactions to verify their integrity before signing, and
offered a tool for it. SwissBorg: it had chosen not to rely on that tool, because it was not
integrated into the dashboard, not open-source or verifiable; and, separately, that the attacker
"did not breach SwissBorg's wallet infrastructure" and the incident "occurred entirely within
Kiln's systems".

The two companies disagree about the decoder, and this book does not referee. Read the lines
they do not dispute. Nobody, on either account, broke into SwissBorg. Its custody system did what
it was built to do: take what the partner's API returned and act on it. OWASP calls this
**unsafe consumption of APIs**, API10: a service gives data from a third-party API more trust than it would give a request body, so a partner that is compromised,
buggy, or simply different from what you assumed writes straight into your state. The request you
validated went out. The response you did not validate came back, and you signed it.

## Where the bug lives: a row from the partner, written into the total

Ledger bills in pounds, but Cedar has customers who pay in euros, so an invoice carries a
`Currency` field. To record a euro invoice in pence, Ledger
needs a rate, and it gets one from a partner feed at `rates.partner.example`, which answers
`GET /latest` with a row like `{"EUR": 0.92}`: pounds per euro. A poller fetches the row every hour
and the store keeps it. Here is the poller and the invoice creation that uses it, before this
chapter's fix.

```go
func (s *Store) vulnerablePollRates() error {
	reply, err := http.Get(partnerRateURL) // default client follows redirects
	if err != nil {
		return err
	}
	defer reply.Body.Close()
	if reply.StatusCode != http.StatusOK {
		return fmt.Errorf("partner status %d", reply.StatusCode)
	}
	var row map[string]float64
	if err := json.NewDecoder(reply.Body).Decode(&row); err != nil {
		return err
	}
	s.rates = row // every key in the partner response is trusted
	return nil
}

func (s *Store) vulnerableCreateInvoice(inv Invoice) Invoice {
	if rate, ok := s.rates[inv.Currency]; ok {
		amount := float64(inv.Amount)
		if inv.Currency == "EUR" {
			amount = inv.AmountEUR * 100
		}
		inv.Amount = int(math.Round(amount * rate))
	}
	s.invoices[inv.ID] = inv
	return inv
}
```

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

```go
type rateRow struct {
	EUR float64 `json:"EUR"`
}
```

```go
func decodeRateRow(body io.Reader) (rateRow, error) {
	var row rateRow
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&row); err != nil {
		return rateRow{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return rateRow{}, fmt.Errorf("extra partner JSON")
	}
	if row.EUR < 0.70 || row.EUR > 1.10 || math.IsNaN(row.EUR) || math.IsInf(row.EUR, 0) {
		return rateRow{}, fmt.Errorf("EUR rate outside accepted band")
	}
	return row, nil
}

func (s *Store) fixedPollRates(ctx context.Context, fetcher outboundFetcher, at time.Time) error {
	reply, err := fetcher.Fetch(ctx, partnerRateURL) // shared egress pins DNS and refuses redirects
	if err != nil {
		s.opsPages++
		return err
	}
	defer reply.Body.Close()
	if reply.StatusCode != http.StatusOK {
		s.opsPages++
		return fmt.Errorf("partner status %d", reply.StatusCode)
	}
	row, err := decodeRateRow(reply.Body)
	if err != nil {
		s.opsPages++
		return err
	}
	s.lastGoodRate = Rate{Value: row.EUR, AsOf: at}
	return nil
}

func (s *Store) fixedCreateInvoice(inv Invoice) Invoice {
	if inv.Currency == "EUR" {
		s.rateReads++ // only invoice creation reads the current partner rate
		inv.FXRate = s.lastGoodRate.Value
		inv.Amount = int(math.Round(inv.AmountEUR * inv.FXRate * 100))
	}
	s.invoices[inv.ID] = inv // GBP never consults the feed
	return inv
}
```

Three details decide whether this is real.

- **The response is decoded into a declared type, with unknown fields rejected.** `rateRow` has one
  field, `EUR`. `DisallowUnknownFields` is the call the property chapter used on `InvoicePatch`;
  here it is pointed the other way, at what comes in from a partner rather than a client. A row
  with a `GBP` key is refused whole, not trimmed.
- **The value is bounded by a range Ledger declares, not by what the partner sent last time.**
  The fixed poller accepts a euro rate only inside `0.70` to `1.10`. Those are Ledger's numbers,
  chosen by whoever owns the business, and they will need revising if the pound ever moves that
  far; the point is that the range is Ledger's, not the feed's. Outside it, the poller keeps the
  last good rate and pages Ops. It does not bill at the new one.
- **The invoice records the rate it used.** `FXRate` and the original `AmountEUR` are stored next
  to the pence. A bad rate that gets through anyway is then a query, not an archaeology, and a
  later quote reads the record rather than the feed.

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
  it, saying it was not integrated, not open-source, not verifiable. Whoever is right, the check
  that would have mattered was one on the consumer's side, in code the consumer could read.
  Validation you cannot inspect is trust with extra steps.
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

```go
func vulnerableChapter10Quote(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, inv, input, ok := chapter10QuoteRequest(w, r, store) // calls LoadInvoiceFor
		if !ok {
			return
		}
		amountPence := int(input.Amount) // GBP requests already carry integer pence
		if inv.Currency == "EUR" {
			amountPence = chapter10Pence(input.Amount, store.rates["EUR"])
		}
		if amountPence <= 0 || amountPence > inv.Amount-inv.Refunded {
			http.Error(w, "invalid quote amount", http.StatusBadRequest)
			return
		}
		quote := store.newQuote(user, inv, amountPence)
		store.noteQuote(user.Tenant, clock())
		writeJSON(w, quote)
	}
}
```

It converts at `store.rates["EUR"]`: today's rate, whatever the poller last stored. Fix the poller
and this is still wrong, in two ways. A bad rate that reached the store before the fix is still
the rate every quote converts at. And even with a perfect feed, a customer billed €300.00 at 0.92
who asks for the full €300.00 back at 0.95 is asking for £285.00 against a £276.00 invoice, and
the remaining-balance check, the one that stops over-refunding, refuses a refund the customer is
plainly owed. Two amounts, converted at two rates, are being compared as if they were one. The
repair is the one the refund chapter made for the quote, applied to the rate: confirm uses the
invoice stored with the quote; quote uses the rate stored with the invoice.

```go
func fixedChapter10Quote(store *Store, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, inv, input, ok := chapter10QuoteRequest(w, r, store) // calls LoadInvoiceFor
		if !ok {
			return
		}
		amountPence := int(input.Amount) // GBP requests already carry integer pence
		if inv.Currency == "EUR" {
			if inv.FXRate == 0 {
				store.opsPages++ // manual reconciliation, never today's rate
				http.Error(w, "rate_reconciliation_required", http.StatusConflict)
				return
			}
			amountPence = chapter10Pence(input.Amount, inv.FXRate)
		}
		if amountPence <= 0 || amountPence > inv.Amount-inv.Refunded {
			http.Error(w, "invalid quote amount", http.StatusBadRequest)
			return
		}
		quote := store.newQuote(user, inv, amountPence)
		store.noteQuote(user.Tenant, clock())
		writeJSON(w, quote)
	}
}
```

The quote stores the pence it computed, so `ConfirmRefund` and the tenant allowance never convert
again. A euro invoice from before this chapter, with no `FXRate` on it, gets no quote until someone
reconciles it by hand. And `store.rates` is now consulted in one place: when an invoice is
created. That is the shape of `LoadInvoiceFor` and `egress`: one door, one check, and every other path
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
  2760000 pence, £27,600.00; 104 untouched. It is the case a compromised feed has no reason to
  send, because a human notices it on the first invoice.
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
