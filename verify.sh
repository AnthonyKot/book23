#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
test_log=$(mktemp "${TMPDIR:-/tmp}/book23-tests.XXXXXX")
trap 'rm -f "$test_log"' EXIT

(
  cd "$repo_dir/service"
  go test -v ./...
) | tee "$test_log"

for group in TestChapter01ExerciseCases TestChapter09ExerciseCases TestChapter09InventoryReconciliation; do
  if ! grep -Fq -- "--- PASS: $group" "$test_log"; then
    echo "missing passing pilot test group: $group" >&2
    exit 1
  fi
done

python3 - "$repo_dir" <<'PY'
import csv
import hashlib
import pathlib
import sys
from html.parser import HTMLParser
from urllib.parse import urlparse

root = pathlib.Path(sys.argv[1])
expected_header = ["claim_id", "claim", "archive", "source_url", "locator", "evidence", "qualification"]

for number in ("00", "01", "09"):
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

class Links(HTMLParser):
    def __init__(self):
        super().__init__()
        self.hrefs = []
    def handle_starttag(self, tag, attrs):
        if tag == "a":
            self.hrefs.extend(value for key, value in attrs if key == "href")

chapter_dir = root / "chapters"
chapter_files = sorted(chapter_dir.glob("*.html")) if chapter_dir.is_dir() else []
if not chapter_files:
    print("chapter/link checks pending: no pilot prose exists yet")
else:
    groups = {
        "opener": list(chapter_dir.glob("00*.html")),
        "chapter 1": list(chapter_dir.glob("01*.html")),
        "chapter 9": list(chapter_dir.glob("09*.html")),
    }
    for label, matches in groups.items():
        if len(matches) != 1:
            raise SystemExit(f"expected one {label} HTML file, found {len(matches)}")
        text = matches[0].read_text(encoding="utf-8")
        if "<h1" not in text or "<title" not in text:
            raise SystemExit(f"missing title structure in {matches[0]}")
        if label != "opener" and "<!--mission-->" not in text:
            raise SystemExit(f"missing exercise marker in {matches[0]}")
    for page in chapter_files:
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

print("claim archives and links verified")
PY

marker_starts=$(rg -n '^// excerpt: ' "$repo_dir/service" -g '*.go' | wc -l)
marker_ends=$(rg -n '^// end excerpt$' "$repo_dir/service" -g '*.go' | wc -l)
if [[ "$marker_starts" -eq 0 || "$marker_starts" -ne "$marker_ends" ]]; then
  echo "unbalanced excerpt markers: $marker_starts starts, $marker_ends ends" >&2
  exit 1
fi

duplicates=$(rg --no-filename -o '^// excerpt: .*' "$repo_dir/service" -g '*.go' | sort | uniq -d)
if [[ -n "$duplicates" ]]; then
  echo "duplicate excerpt marker names:" >&2
  echo "$duplicates" >&2
  exit 1
fi

echo "Lane A verification passed"
