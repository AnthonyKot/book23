#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
test_log=$(mktemp "${TMPDIR:-/tmp}/book23-tests.XXXXXX")
trap 'rm -f "$test_log"' EXIT

(
  cd "$repo_dir/service"
  go test -v ./...
) | tee "$test_log"

for group in TestChapter01ExerciseCases TestChapter01RegisteredInvoiceRoutesCovered TestChapter01RefundSequence TestChapter02ExerciseCases TestChapter02SessionLifetimeAndLogin TestChapter02EarlierRepairAndRouteIdentity TestChapter02CredentialBoundary TestChapter03ExerciseReadCases TestChapter03ExercisePatchCases TestChapter03MarshalGuardAndEarlierRepairs TestChapter03SessionExpiryAndAllReadViews TestChapter04OTPExerciseCases TestChapter04LookupExerciseCases TestChapter04PageExerciseCases TestChapter04SharedHandlerAndEarlierRepairs TestChapter05ExerciseCases TestChapter05AdminVoidAndScopeOrder TestEveryRouteDeclares TestChapter05MissingDeclarationFailsRegistration TestChapter05EarlierRepairs TestChapter06ExerciseSequences TestChapter06ReminderExerciseAndEarlierRepairs TestChapter06AllowanceDayAndAtomicity TestChapter07WebhookExerciseCases TestChapter07LinkLocalDialAndPinning TestChapter07AddressClasses TestChapter07PDFSecondPathAndEarlierRepairs TestChapter07PDFLinkLocalAnswer TestChapter08ExerciseCases TestChapter08ProductionSettings TestChapter08EarlierRepairs TestChapter09ExerciseCases TestChapter09InventoryReconciliation; do
  if ! grep -Fq -- "--- PASS: $group" "$test_log"; then
    echo "missing passing pilot test group: $group" >&2
    exit 1
  fi
done

python3 - "$repo_dir" <<'PY'
import csv
import hashlib
import pathlib
import re
import sys
from html.parser import HTMLParser
from urllib.parse import urlparse

root = pathlib.Path(sys.argv[1])
expected_header = ["claim_id", "claim", "archive", "source_url", "locator", "evidence", "qualification"]

claim_archives = set()
for number in (f"{i:02d}" for i in range(11)):
    claim_file = root / "checks" / "claims" / f"{number}.tsv"
    with claim_file.open(newline="", encoding="utf-8") as handle:
        rows = list(csv.reader(handle, delimiter="\t"))
    if not rows or rows[0] != expected_header or len(rows) < 2:
        raise SystemExit(f"invalid or empty claim ledger: {claim_file}")
    for line, row in enumerate(rows[1:], 2):
        if len(row) != len(expected_header):
            raise SystemExit(f"{claim_file}:{line}: expected {len(expected_header)} fields")
        archive = root / row[2]
        if not archive.is_file() or archive.stat().st_size == 0:
            raise SystemExit(f"{claim_file}:{line}: missing archive {row[2]}")
        claim_archives.add(row[2])
        parsed = urlparse(row[3])
        if parsed.scheme != "https" or not parsed.netloc:
            raise SystemExit(f"{claim_file}:{line}: invalid source URL {row[3]}")

source_file = root / "resources" / "incidents" / "SOURCES.tsv"
with source_file.open(newline="", encoding="utf-8") as handle:
    sources = list(csv.DictReader(handle, delimiter="\t"))
for row in sources:
    archive = root / row["archive"]
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    if digest != row["sha256"]:
        raise SystemExit(f"source checksum mismatch: {row['archive']}")
manifest_archives = {row["archive"] for row in sources}
if unregistered := claim_archives - manifest_archives:
    raise SystemExit(f"claim archives absent from SOURCES.tsv: {sorted(unregistered)}")

for number in ("01", "02", "03", "04", "05", "06", "07", "08", "09"):
    chapters = list((root / "chapters").glob(f"{number}-*.md"))
    if len(chapters) != 1 or "{{excerpt:" in chapters[0].read_text(encoding="utf-8"):
        raise SystemExit(f"built chapter {number} still has an excerpt placeholder")

ch02 = next((root / "chapters").glob("02-*.md")).read_text(encoding="utf-8")
code = (root / "service" / "ch02_auth.go").read_text(encoding="utf-8")
for name in ("ch02-vulnerable", "ch02-fixed"):
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch02:
        raise SystemExit(f"Chapter 02 printed excerpt {name} differs from marked Go source")

ch03 = next((root / "chapters").glob("03-*.md")).read_text(encoding="utf-8")
for name in ("ch03-vulnerable-read", "ch03-view", "ch03-vulnerable-patch", "ch03-patch", "ch03-marshal-guard"):
    file = "model.go" if name == "ch03-marshal-guard" else "ch03_properties.go"
    code = (root / "service" / file).read_text(encoding="utf-8")
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch03:
        raise SystemExit(f"Chapter 03 printed excerpt {name} differs from marked Go source")

ch04 = next((root / "chapters").glob("04-*.md")).read_text(encoding="utf-8")
code = (root / "service" / "ch04_limits.go").read_text(encoding="utf-8")
for name in ("ch04-otp-vulnerable", "ch04-limiter-per-ip", "ch04-otp-fixed"):
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch04:
        raise SystemExit(f"Chapter 04 printed excerpt {name} differs from marked Go source")

ch05 = next((root / "chapters").glob("05-*.md")).read_text(encoding="utf-8")
for name, file in (("ch05-vulnerable-delete", "authz.go"), ("ch05-access-decl", "routes.go"), ("ch05-require-access", "authz.go")):
    code = (root / "service" / file).read_text(encoding="utf-8")
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch05:
        raise SystemExit(f"Chapter 05 printed excerpt {name} differs from marked Go source")
test_code = (root / "service" / "ch05_test.go").read_text(encoding="utf-8")
if not re.search(r"(?m)^// excerpt: ch05-declares-test\n(.*?)^// end excerpt$", test_code, re.S | re.M):
    raise SystemExit("Chapter 05 declaration test marker missing")

ch06 = next((root / "chapters").glob("06-*.md")).read_text(encoding="utf-8")
code = (root / "service" / "ch06_flow.go").read_text(encoding="utf-8")
for name in ("ch06-vulnerable-confirm", "ch06-fixed-confirm", "ch06-per-user-counter", "ch06-tenant-allowance"):
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch06:
        raise SystemExit(f"Chapter 06 printed excerpt {name} differs from marked Go source")

ch07 = next((root / "chapters").glob("07-*.md")).read_text(encoding="utf-8")
for name, file in (("ch07-webhook-vulnerable", "ch07_handlers.go"), ("ch07-egress-policy", "ch07_egress.go"), ("ch07-webhook-fixed", "ch07_handlers.go"), ("ch07-pdf-logo-fetch", "ch07_handlers.go")):
    code = (root / "service" / file).read_text(encoding="utf-8")
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch07:
        raise SystemExit(f"Chapter 07 printed excerpt {name} differs from marked Go source")

ch08 = next((root / "chapters").glob("08-*.md")).read_text(encoding="utf-8")
for name, file in (("ch08-vulnerable-health-decl", "ch08_settings.go"), ("ch08-fixed-health-decl", "ch08_settings.go"), ("ch08-debug-error-writer", "ch08_settings.go"), ("ch08-production-settings", "ch08_settings.go"), ("ch08-settings-test", "ch08_test.go")):
    code = (root / "service" / file).read_text(encoding="utf-8")
    match = re.search(r"(?m)^// excerpt: " + name + r"\n(.*?)^// end excerpt$", code, re.S | re.M)
    if not match or "```go\n" + match.group(1).rstrip("\n") + "\n```" not in ch08:
        raise SystemExit(f"Chapter 08 printed excerpt {name} differs from marked Go source")

class Links(HTMLParser):
    def __init__(self):
        super().__init__()
        self.hrefs = []
    def handle_starttag(self, tag, attrs):
        if tag == "a":
            self.hrefs.extend(value for key, value in attrs if key == "href")

chapter_dir = root / "docs" / "chapters"
sources = sorted((root / "chapters").glob("*.md"))
chapter_files = sorted(chapter_dir.glob("*.html")) if chapter_dir.is_dir() else []
if not chapter_files:
    print("chapter/link checks pending: run `node site/build.mjs` to render docs/")
else:
    rendered = {page.stem for page in chapter_files}
    for source in sources:
        if source.stem not in rendered:
            raise SystemExit(f"chapter {source.name} has no rendered page; run node site/build.mjs")
    for page in chapter_files:
        text = page.read_text(encoding="utf-8")
        if "<h1" not in text or "<title" not in text:
            raise SystemExit(f"missing title structure in {page}")
        if not page.name.startswith("00") and "<!--mission-->" not in text:
            raise SystemExit(f"missing exercise marker in {page}")
        source = root / "chapters" / (page.stem + ".md")
        if "{{excerpt:" in text:
            raise SystemExit(f"unrendered placeholder in {page}")
    for page in chapter_files + [root / "docs" / "index.html"]:
        parser = Links()
        parser.feed(page.read_text(encoding="utf-8"))
        for href in parser.hrefs:
            if href.startswith(("#", "mailto:")):
                continue
            parsed = urlparse(href)
            if parsed.scheme:
                if parsed.scheme not in ("http", "https") or not parsed.netloc:
                    raise SystemExit(f"invalid external link in {page}: {href}")
                continue
            target = (page.parent / parsed.path).resolve()
            if not target.exists():
                raise SystemExit(f"broken local link in {page}: {href}")
    pending = sum(page.read_text(encoding="utf-8").count("excerpt-pending") for page in chapter_files)
    print(f"{len(chapter_files)} chapter pages checked; {pending} code blocks still pending")

print("claim archives and links verified")
PY

marker_starts=$(grep -rn '^// excerpt: ' "$repo_dir/service" --include='*.go' | wc -l)
marker_ends=$(grep -rn '^// end excerpt$' "$repo_dir/service" --include='*.go' | wc -l)
if [[ "$marker_starts" -eq 0 || "$marker_starts" -ne "$marker_ends" ]]; then
  echo "unbalanced excerpt markers: $marker_starts starts, $marker_ends ends" >&2
  exit 1
fi

duplicates=$(grep -rho '^// excerpt: .*' "$repo_dir/service" --include='*.go' | sort | uniq -d)
if [[ -n "$duplicates" ]]; then
  echo "duplicate excerpt marker names:" >&2
  echo "$duplicates" >&2
  exit 1
fi

echo "Lane A verification passed"
