# Chapter 1 incident choice — API1 BOLA

Compared original or independently verified first-hand accounts on 2026-09-23. The Chapter 1
incident must support the narrow claim that identifying another person's object in a request
exposed data without an object-level authorization check. Ledger's invoice code is an analogy,
not a reconstruction of any real service.

| Case and source | Mechanism sentence from the record | Fit for Chapter 1 |
|---|---|---|
| [Peloton, Jan Masters (2021)](https://www.pentestpartners.com/security-blog/tour-de-peloton-exposed-user-data/) | A client could manually change the workout IDs in `POST /stats/workouts/details`; after the endpoint began requiring login, an authenticated member could still receive other members' information, including information from private profiles. | **Selected.** The editable `ids` request and failed per-object check are visible in the researcher's technical account. This is an exposure case; do not claim the researcher changed a record or knew the server's implementation. |
| [USPS Informed Visibility, Brian Krebs (2018)](https://krebsonsecurity.com/2018/11/usps-site-exposed-data-on-60-million-users/) | Krebs independently confirmed that any logged-in USPS user could query other users' account details through search parameters, including wildcards; login did not constrain which results the account could read. | Strong backup with broader impact, but the documented request is a search, not a simple ID lookup; the original researcher was anonymous and the published account gives less request detail. Do not imply that all ~60 million records were actually downloaded. |
| [Tinder, Max Veytsman (2014)](https://blog.includesecurity.com/2014/02/how-i-was-able-to-track-the-location-of-any-tinder-user/) | The user-by-ID response for potential matches exposed a highly precise `distance_mi` value; repeated queries from spoofed locations let the researcher infer a user's location. | **Not Chapter 1.** The decisive flaw is the precision of a returned property and location privacy, rather than proof that the caller was unauthorized to view the object. It may fit a later property-exposure discussion. |

Decision: use Peloton for Chapter 1, subject to the normal claim gate. Cite the specific request,
what the researcher actually observed, and the partial login fix. If drafting reveals a claim the
source cannot support, narrow it or return to USPS. Do not conflate Peloton's three reported
API issues into one endpoint or present Ledger's `/v1` and invoice behavior as Peloton facts.
