// One row per chapter. `file` is the Markdown source in chapters/; the H1 there is the title.
// `status` is the register state from CONTEXT.md §4 and is shown on the page so a reader of a
// draft knows what they are reading. Keep this list in the book's order.
export const book = {
  name: "Security Rebook",
  tagline: "ten ways an API lets the wrong request through",
  description:
    "Eleven chapters, one small Go service, ten real break-ins. Each chapter shows the check a real API skipped, rebuilds it on Ledger, fixes it, and then finds the second door.",
};

export const chapters = [
  { file: "00-the-lookup-that-never-asks", kicker: "Opener", cls: "The question", incident: "Coinbase, 2022", payoff: "Every request assumes a check already happened. Find which one.", status: "drafted" },
  { file: "01-whos-asking", kicker: "API1", cls: "Broken object-level authorization", incident: "Peloton, 2021", payoff: "Login says who is calling. It never says whether they may see this record.", status: "pilot, drafted" },
  { file: "02-one-key-for-every-door", kicker: "API2", cls: "Broken authentication", incident: "BrewDog, 2021", payoff: "A key that every copy of the app carries identifies the app, not the person.", status: "drafted" },
  { file: "03-the-row-you-didnt-mean-to-send", kicker: "API3", cls: "Broken object property level authorization", incident: "DPD, 2022", payoff: "The handler returned the row. The row had more in it than the page ever showed.", status: "drafted" },
  { file: "04-nobody-counted", kicker: "API4", cls: "Unrestricted resource consumption", incident: "Instagram, 2019", payoff: "A limit keyed on the wrong thing is a limit on how many of that thing the attacker can afford.", status: "drafted" },
  { file: "05-same-door-different-verb", kicker: "API5", cls: "Broken function level authorization", incident: "Dealer portal, 2025", payoff: "Same URL, different verb, same check. The page was hidden; the function was not.", status: "drafted" },
  { file: "06-every-request-was-valid", kicker: "API6", cls: "Unrestricted access to sensitive business flows", incident: "Ticketmaster bots, 2021 complaint", payoff: "Every request was valid. The harm was the volume, and the limit was on the wrong object.", status: "drafted" },
  { file: "07-the-server-that-fetched-for-you", kicker: "API7", cls: "Server side request forgery", incident: "Shopify Exchange, 2018", payoff: "The string was checked. The socket connected somewhere else.", status: "drafted" },
  { file: "08-left-on", kicker: "API8", cls: "Security misconfiguration", incident: "Power Apps portals, 2021", payoff: "The check was there, wired correctly, and switched off.", status: "drafted" },
  { file: "09-deprecated-is-a-label", kicker: "API9", cls: "Improper inventory management", incident: "Optus, 2022", payoff: "A host that is not on your list is a host your fixes do not reach.", status: "pilot, drafted" },
  { file: "10-what-you-swallowed", kicker: "API10", cls: "Unsafe consumption of APIs", incident: "Kiln and SwissBorg, 2025", payoff: "The request you validated went out. The response you did not validate came back.", status: "drafted" },
];
