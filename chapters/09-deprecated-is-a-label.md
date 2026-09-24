# Deprecated Is a Label

<!-- claims to gate in checks/claims/09.tsv (all are ACMA's allegations in the redacted concise statement annexed to the Federal Court order of 19 June 2024, VID429/2024; none is an adjudicated finding; redacted system names stay unfilled): 17–20 September 2022 relevant period; around 3.6 million active customers vs more than 9.5 million former and current customers (keep distinct); data fields listed in ¶2; Main Domain www.optus.com.au and Target Domain api.www.optus.com.au; Target Domain internet-facing from 20 April 2017, intended to segregate API traffic from static content; dormant and not in use since 2017, not decommissioned until after the attack; three Target APIs meant to return information only after authentication; September 2018 coding error in one access control, ineffective for both domains; both internet-facing with the error from June 2020; August 2021 Main fixed, Target not; "not highly sophisticated", "trial and error"; attacker bypassed access controls and sent requests to the Target APIs; first detected 17 September, aware ~8pm 19 September, internet traffic blocked ~3:45am 20 September, decommissioned 21 September 2022. Source: resources/incidents/09/optus-filed-order-and-concise-statement.pdf. -->

Between 17 and 20 September 2022 someone pulled the personal records of more than 9.5 million
current and former customers out of Optus, Australia's second-largest telecommunications company:
names, email addresses, dates of birth and phone numbers, and for some of them the home address
and the driver's licence, passport or Medicare number that Australians use to prove who they
are. What follows is the account the
regulator gave a court two years later. The Australian Communications and Media Authority, ACMA,
sued Optus in the Federal Court in May 2024, and the redacted concise statement it filed is public.
Everything below is what ACMA alleges. Optus's systems are named in the filing only as black bars,
and this chapter leaves the bars where they are.

ACMA's account is about two domains. Customers could reach their information through APIs on
`www.optus.com.au`, which the filing calls the Main Domain, and on `api.www.optus.com.au`, the
Target Domain. The Target Domain had been put on the internet in April 2017 to take API traffic
away from the static content on the main site. It had three APIs that were meant to return a
customer's information only after that customer had authenticated. And, ACMA alleges, it had been
dormant and not in use since 2017. Nobody needed it. It was not decommissioned.

In September 2018 a coding error was made in one of the access controls, and the error made that
control ineffective for both domains. By June 2020 both domains were internet-facing with the
error in them. In August 2021 Optus detected that the Main Domain was vulnerable because of the
error and fixed it there. It did not detect, and did not fix, the same issue on the Target
Domain. The filing says the Target Domain "was permitted to sit dormant and vulnerable to attack
for two years and was not decommissioned despite the lack of any need for it."

The attack, in ACMA's words, "was not highly sophisticated" and "was carried out through a simple
process of trial and error": the attacker bypassed the access controls and sent requests to the
Target APIs, which returned customers' personal information. Optus first detected it on 17
September, understood at about 8pm on 19 September that an attack was under way, blocked internet
traffic to the Target Domain at about 3:45am on 20 September, and decommissioned it on 21
September. ACMA is seeking penalties in respect of at least 3.6 million of those customers, the
ones who were active subscribers at the time.

Set aside the coding error for a moment; it was fixed, in the place people were looking. The
subject of this chapter is the place they were not. OWASP calls it **improper inventory
management**, ninth in its API Security Top 10, and the class is easy to misread as
housekeeping. It is not. A host that is not on your list is a host your fixes do not reach.

## The route you kept, and the host you never listed

Ledger's version of this begins where the BOLA chapter ended. `/v1/invoices/{id}` and its PDF
route were found, put behind `LoadInvoiceFor` like everything else, and left running because an
old version of the mobile app still calls them. Ops wrote "deprecated" next to both in the
inventory and the plan was to switch them off once usage dropped. The build tested in this chapter
starts from that state: Alice already gets 404 for Birch's invoice 205 on every route. That
repair is not in question here.

Ops's inventory has one host on it, `api.ledger.example`. The gateway in front of the service
knows two:

```go
// ch09-gateway-host-filter
func chapter09Gateway(mode Mode, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == PublicHost {
			next.ServeHTTP(w, r)
			return
		}
		if mode == Vulnerable && host == StagingHost && strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}
```

`StagingHost` is `ledger-staging.internal`. It was set up when v1 was the only API, so the mobile
team could test against a name that would not change, and the rule forwards anything under `/v1/`
to the same service. The name says internal; the gateway does not check where the request came
from. Nobody on the current team knows the rule exists, because the two places a person would
look do not show it. The code's route table lists paths, not hosts:

```go
// ch09-code-route-fixture
var vulnerableCodeRoutes = []Route{
	{Method: "GET", Pattern: "/v2/invoices/{id}"},
	{Method: "GET", Pattern: "/v2/me/invoices"},
	{Method: "GET", Pattern: "/v1/invoices/{id}"},
	{Method: "GET", Pattern: "/v1/invoices/{id}/pdf"},
}

var fixedCodeRoutes = vulnerableCodeRoutes[:2]
```

And the inventory document lists what Ops wrote down. Grepping for `mux.Handle`, the recipe from
the BOLA chapter, finds every route the code registers and says nothing about which names the
gateway answers to. The staging host is in the third list, gateway configuration, and in a fourth,
live traffic, and in neither of the two anyone reads.

That is the shape of ACMA's allegation. The Target Domain was not a secret. It had a name, a
purpose and a date it went live. It had simply stopped being anywhere that a fix would be applied.
When the access control was repaired on the Main Domain, the repair went where the report pointed,
and the domain nobody used did not get it.

## Why "deprecated" protects nothing

Deprecated is a word in a document. The gateway does not read the document. Until the route
stops answering, deprecated means exactly what it meant the day before: the route serves whatever
it served, through whatever hosts forward to it, with whatever controls happen to have reached it.

Here is how that goes wrong at Ledger, not with this chapter's repair but with the next one. The
chapter on resource consumption put a thirty-a-minute budget on the lookup routes, at the gateway
and again in the shared handler. The gateway copy was written against the hosts Ops listed, so
`ledger-staging.internal` never got it. The handler copy saved the day: the budget sits where
every request passes, so the staging host is metered anyway, and the route shown here still calls
`LoadInvoiceFor`, so Alice still cannot read Birch's invoice that way. Nothing leaks today. But
the reason nothing leaks is that two earlier chapters happened to put their checks inside the
handler. The next control that lives only at the gateway, a host-level policy, a certificate
rule, a log that feeds the alerting, will be applied to Ops's inventory, and this host is not in
it. Optus's Target Domain did not need to be exploitable in 2017 to be the way in five years
later.

So the question the BOLA chapter asked, "which routes can reach this data?", was one list short.
Here it is with the missing one, as code rather than a checklist:

```go
// ch09-reconcile-inventory
func ReconcileInventory(f InventoryFixture) []string {
	findings := make([]string, 0)
	findings = append(findings, unlistedHostFindings(f)...)
	findings = append(findings, deprecatedSurfaceFindings(f)...)
	findings = append(findings, configurationFindings(f)...)
	sort.Strings(findings)
	return findings
}
```

The fixture holds four lists: the hosts Ops declared, the routes the code registers, the surfaces
the gateway forwards (host, method, route), and the surfaces seen in traffic. Three comparisons
fall out. A host in gateway configuration or traffic that is not declared is an unlisted host. A
surface marked deprecated that still appears in the gateway or in traffic is still serving. A
gateway surface with no code behind it, traffic with no gateway rule, or a code route no host
exposes is configuration drift. Run against Ledger before this chapter's fix, it prints four
lines:

```text
deprecated-still-serving:GET api.ledger.example/v1/invoices/{id}
deprecated-still-serving:GET api.ledger.example/v1/invoices/{id}/pdf
deprecated-still-serving:GET ledger-staging.internal/v1/invoices/{id}
unlisted-host:ledger-staging.internal
```

The first two are the deprecation everyone knew about, stated as a fact about the gateway rather
than a note in a document. The third and fourth are the host nobody knew about. Notice that the
deprecated list did not need to mention the staging host to catch it: the unlisted-host check
catches any name that forwards, and the deprecated check then catches every surface on it that
matches a retired route. Ops only had to write down the hosts it meant to have.

## Retired means it stops answering

The fix has two halves, and the second is the one that gets skipped.

The first half is a date. "Once usage drops" is not one. Ops reads the traffic list, finds the
callers still on `/v1`, and tells the mobile team which app versions they are. The last of them
gets a cut-off. On that date the gateway stops forwarding the routes, the code stops registering
them, and a probe from outside gets 404 on both hosts. The fixed build of Ledger is that state:
`fixedCodeRoutes` is the first two entries of the vulnerable list, the gateway forwards the
public host only, and `ReconcileInventory` prints nothing.

The second half is the inventory itself. It has to be produced from the gateway and the traffic,
not typed by hand, and it has to be re-run: in the build, or on a schedule, or both. The
reconciliation above is a test in `service/ch09_test.go`, and it asserts the four findings against
the vulnerable build and none against the fixed one. A host that appears in the gateway config
next year will fail it. That is the property Optus lacked, on ACMA's account: a domain could be
put on the internet in 2017, fall out of use the same year, and still be nowhere that the 2021
fix would look.

Two limits, so the fixture is not mistaken for more than it is. Ledger's gateway is a function in
the same process, and its traffic list is a slice; a real service gets those from the gateway's
configuration repository and its access logs, and the test proves only that the comparison is
done, not that the logs are complete. And the reconciliation cannot find a host that neither the
gateway nor the logs mention. For that there is no substitute for probing your own names from
outside: DNS records you own, certificates issued for them, and a request to each one.

<!--mission-->
## Exercise: which list is it on?

Use Ledger before this chapter's fix: the public host is `api.ledger.example`, the gateway also
forwards `/v1/` paths on `ledger-staging.internal`, `/v1` is marked deprecated, and every invoice
route already goes through `LoadInvoiceFor`. Alice, Cedar, is the caller throughout; invoice 104
is Cedar's and 205 is Birch's. Six requests:

```text
A. api.ledger.example       GET /v2/invoices/104
B. api.ledger.example       GET /v2/invoices/205
C. api.ledger.example       GET /v1/invoices/104
D. ledger-staging.internal  GET /v1/invoices/104
E. ledger-staging.internal  GET /v2/invoices/104
F. ledger-preview.internal  GET /v1/invoices/104
```

For each request: which of the four lists is its surface on (declared hosts, code routes, gateway,
traffic), what does the gateway do with it, and what status comes back before the fix and after?
Then say which finding, if any, `ReconcileInventory` prints for it.

**Check your answer.**

- **A: 200 before and after.** The host is declared, the gateway forwards it, the route is in the
  code, the loader finds invoice 104 with Alice's tenant. No finding: this is the current API and it
  is in every list. It is the decoy only if you thought this chapter retires everything old.
- **B: 404 before and after.** Same surface as A; the difference is in the loader, not the gateway.
  `LoadInvoiceFor` sees Birch on the stored invoice and Cedar on Alice and returns not-found. The
  BOLA repair is unchanged by retiring v1; the two chapters fix different lists.
- **C: 200 before, 404 after.** Declared host, gateway forwards, code registers the route, and
  the loader returns Cedar's own invoice. Before the fix this is the deprecation everyone knew
  about, and the reconciliation prints `deprecated-still-serving` for it. After the fix the route is
  gone from the code and from the gateway, so the answer is 404 with no invoice.
- **D: 200 before, 404 after.** The host is on no declared list, but the gateway rule forwards
  `/v1/` on it to the same mux, and the mux has the route. Alice gets her own invoice through a
  name Ops has never heard of. Two findings: `unlisted-host` for the name, and
  `deprecated-still-serving` for the surface. After the fix the gateway answers 404 for the host
  regardless of path.
- **E: 404 before and after.** This is D with one character changed, and the outcome flips. The
  gateway's staging rule matches the path prefix `/v1/`, so `/v2/invoices/104` on the staging host
  is refused at the gateway before any route or loader runs. If you gave E a 200 because the
  staging host is open, you inferred the rule from its name. It forwards a prefix, not a host, and
  the reconciliation reports the surfaces it actually forwards, not the host in general.
- **F: 404 before and after.** The host looks like D and is on none of the four lists, including
  the gateway's. The gateway falls through to not-found. No finding, because the reconciliation
  works from what forwards and what is seen; a name that does nothing is invisible to it. That is
  the limit above, and the reason outside probing still has a job.

If you marked E a 200, the lesson is the one the chapter is about: you cannot tell what a host
serves from what it is called. Read the rule.

*Incident from the Australian Communications and Media Authority's redacted concise statement
annexed to the Federal Court of Australia's order of 19 June 2024 in ACMA v Optus Mobile Pty
Limited, VID429/2024, as filed allegations. The method follows Colin Domoney, Defending APIs
(Packt, 2024).*
