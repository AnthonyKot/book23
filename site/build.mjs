// Renders chapters/*.md into docs/ as a static site (GitHub Pages), following the book11 shell.
// Usage: node site/build.mjs
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { marked } from "marked";
import { book, chapters } from "./catalog.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "..");
const out = path.join(root, "docs");

// docs/ also holds project documents (HANDOVER.md, CH01-INCIDENT-CHOICE.md); remove only what
// this script generates.
for (const generated of ["chapters", "assets", "index.html", "404.html", ".nojekyll"]) {
  fs.rmSync(path.join(out, generated), { recursive: true, force: true });
}
for (const directory of [out, path.join(out, "chapters"), path.join(out, "assets")]) {
  fs.mkdirSync(directory, { recursive: true });
}

const escapeHtml = (value) => String(value)
  .replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;");

const slugify = (value) => value.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/(^-|-$)/g, "");

const renderer = new marked.Renderer();
renderer.heading = function ({ tokens, depth }) {
  const rendered = this.parser.parseInline(tokens);
  const id = slugify(rendered.replace(/<[^>]+>/g, ""));
  return `<h${depth} id="${id}">${rendered}</h${depth}>\n`;
};
renderer.table = function (token) {
  const table = marked.Renderer.prototype.table.call(this, token);
  if (token.header.length <= 3) return table;
  return `<div class="table-scroll" role="region" aria-label="Table; scroll horizontally if needed" tabindex="0">${table}</div><p class="table-hint">Scroll the table sideways to see all columns.</p>`;
};
renderer.link = function ({ href, title, tokens }) {
  const external = /^https?:\/\//.test(href);
  const attributes = `${title ? ` title="${escapeHtml(title)}"` : ""}${external ? ' target="_blank" rel="noreferrer"' : ""}`;
  return `<a href="${escapeHtml(href)}"${attributes}>${this.parser.parseInline(tokens)}</a>`;
};
marked.setOptions({ gfm: true, renderer });

// Excerpt placeholders are filled by Lane A from marked blocks in service/. Until then they
// render as a visible pending box so a reader of a draft never mistakes a gap for prose.
function replacePlaceholders(markdown, slug) {
  const pending = [];
  const replaced = markdown.replace(/^\{\{excerpt:([a-z0-9-]+)\}\}[ \t]*$/gm, (_, name) => {
    pending.push(name);
    return `<div class="excerpt-pending" role="note"><span>Code pending</span><code>${name}</code> — this Go block is built from the chapter's brief and will appear here once the service code exists.</div>\n`;
  });
  return { markdown: replaced, pending };
}

function readChapter(entry) {
  const source = fs.readFileSync(path.join(root, "chapters", `${entry.file}.md`), "utf8");
  const title = (source.match(/^# (.+)$/m) || [, entry.file])[1].trim();
  const number = entry.file.slice(0, 2);
  return { ...entry, source, title, number, slug: entry.file };
}

const items = chapters.map(readChapter);

function navItems(prefix, activeSlug = "") {
  return items.map((item, index) => `
    <a class="book-nav__item${item.slug === activeSlug ? " is-active" : ""}" href="${prefix}chapters/${item.slug}.html"${index === 0 ? "" : ` data-mission-link="${item.slug}"`}>
      <span class="book-nav__number">${item.number}</span>
      <span><small>${escapeHtml(item.kicker)}</small>${escapeHtml(item.title)}</span>
      <span class="book-nav__check" aria-label="Exercise completed">✓</span>
    </a>`).join("");
}

function shell({ title, description, prefix = "", activeSlug = "", body, pageClass = "" }) {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="description" content="${escapeHtml(description)}">
  <meta name="theme-color" content="#1f2a44">
  <meta property="og:title" content="${escapeHtml(title)}">
  <meta property="og:description" content="${escapeHtml(description)}">
  <meta property="og:type" content="website">
  <title>${escapeHtml(title)} · ${escapeHtml(book.name)}</title>
  <link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'%3E%3Crect width='64' height='64' rx='18' fill='%231f2a44'/%3E%3Ctext x='16' y='45' font-size='36' fill='%23fffdf8'%3ES%3C/text%3E%3C/svg%3E">
  <link rel="stylesheet" href="${prefix}assets/styles.css">
  <script defer src="${prefix}assets/app.js"></script>
</head>
<body class="${pageClass}" data-active-slug="${escapeHtml(activeSlug)}">
  <a class="skip-link" href="#main">Skip to content</a>
  <div class="reading-progress" aria-hidden="true"><span></span></div>
  <header class="site-header">
    <a class="wordmark" href="${prefix}index.html" aria-label="${escapeHtml(book.name)} home">
      <span class="wordmark__mark">S<span>?</span></span>
      <span><strong>${escapeHtml(book.name)}</strong><small>${escapeHtml(book.tagline)}</small></span>
    </a>
    <div class="header-actions">
      <span class="progress-summary" data-progress-summary>0 of ${items.length - 1} exercises</span>
      <button class="menu-button" type="button" data-menu-button aria-expanded="false" aria-controls="book-navigation">Contents</button>
    </div>
  </header>
  <div class="page-shell">
    <aside class="book-nav" id="book-navigation" data-book-nav>
      <div class="book-nav__intro">
        <a href="${prefix}index.html">Contents</a>
        <p>${items.length} chapters. One service, attacked in order.</p>
      </div>
      <nav aria-label="Book contents">${navItems(prefix, activeSlug)}</nav>
      <a class="book-nav__skips" href="${prefix}index.html#how">How this book works →</a>
    </aside>
    ${body}
  </div>
</body>
</html>`;
}

function chapterPage(item, index) {
  const { markdown, pending } = replacePlaceholders(item.source, item.slug);
  let article = marked.parse(markdown);
  article = article.replace(/^<h1[^>]*>.*?<\/h1>\s*/s, "");
  const missionOpen = `<section class="mission" data-mission="${item.slug}"><div class="mission__label">Exercise</div>`;
  if (article.includes("<!--mission-->")) {
    // Keep the marker in the output: verify.sh looks for it.
    article = article.replace("<!--mission-->", `<!--mission-->${missionOpen}`);
    article += "</section>";
  }
  const previous = items[index - 1];
  const next = items[index + 1];
  const pager = `<nav class="essay-pager" aria-label="Adjacent chapters">
    ${previous ? `<a href="${previous.slug}.html"><span>Previous</span>${escapeHtml(previous.title)}</a>` : "<span></span>"}
    ${next ? `<a class="essay-pager__next" href="${next.slug}.html"><span>Next</span>${escapeHtml(next.title)}</a>` : `<a class="essay-pager__next" href="../index.html"><span>Return</span>Contents</a>`}
  </nav>`;
  const draftNote = pending.length
    ? `<div class="currency-note"><strong>Draft</strong>Status: ${escapeHtml(item.status)}. ${pending.length} code block${pending.length === 1 ? "" : "s"} still pending from the service; the prose describes what each will show.</div>`
    : `<div class="currency-note"><strong>Draft</strong>Status: ${escapeHtml(item.status)}. Every Go block on this page is an excerpt of <code>service/</code> and its exercise table runs as a test in both builds.</div>`;
  const exercise = index === 0 ? "" : `<div class="mission-action" data-mission-action="${item.slug}">
      <div><strong>Check your understanding.</strong><span>Work the exercise before reading the answers, then mark it done.</span></div>
      <button type="button" data-complete-mission="${item.slug}">Mark exercise complete</button>
    </div>`;
  const body = `<main id="main" class="essay-page">
    <header class="essay-hero">
      <div class="essay-kicker"><span>${item.number}</span>${escapeHtml(item.kicker)} · ${escapeHtml(item.cls)}</div>
      <h1 class="essay-title">${escapeHtml(item.title)}</h1>
      <p class="essay-book">Incident: ${escapeHtml(item.incident)}</p>
      <p class="essay-payoff">${escapeHtml(item.payoff)}</p>
    </header>
    ${draftNote}
    <article class="prose">${article}</article>
    ${exercise}
    ${pager}
  </main>`;
  return shell({ title: item.title, description: item.payoff, prefix: "../", activeSlug: item.slug, body, pageClass: "article-view" });
}

function homePage() {
  const cards = items.map((item) => `<article class="shelf-card${item.number === "01" ? " shelf-card--recommended" : ""}" data-mission-card="${item.slug}">
    <div class="shelf-card__top"><span>${item.number} · ${escapeHtml(item.kicker)}</span><span class="shelf-card__status">Unread</span></div>
    <h3><a href="chapters/${item.slug}.html">${escapeHtml(item.title)}</a></h3>
    <p>${escapeHtml(item.payoff)}</p>
    <div class="shelf-card__book"><cite>${escapeHtml(item.incident)}</cite><span>${escapeHtml(item.cls)}</span></div>
    <a class="shelf-card__action" href="chapters/${item.slug}.html">Read the chapter <span>→</span></a>
  </article>`).join("");
  const body = `<main id="main" class="home-page">
    <section class="home-hero">
      <div class="home-hero__eyebrow">Real break-ins, one small service</div>
      <h1>Which check did<br><em>this request skip?</em></h1>
      <p>${escapeHtml(book.description)}</p>
      <p class="home-why">The incidents are told from the original public record. The service, Ledger, is invented, written in Go with nothing but the standard library, and every fix on these pages is a test that runs in both the vulnerable and the fixed build.</p>
      <div class="home-hero__actions">
        <a class="button button--primary" href="chapters/${items[0].slug}.html">Start with the opener</a>
        <a class="button button--quiet" href="#the-shelf">Browse ${items.length} chapters</a>
      </div>
      <div class="home-proof"><span><strong>${items.length - 1}</strong> incidents</span><span><strong>1</strong> service</span><span><strong>10</strong> OWASP API classes</span></div>
    </section>
    <section class="shelf-section" id="the-shelf">
      <div class="section-heading"><div><span class="section-label">Contents</span><h2>Ten doors, in the order OWASP lists them</h2></div><p>Read them in order: each chapter's fix stays in force for the next, and each new bug is found by trying the earlier repair against a new route.</p></div>
      <div class="shelf-grid">${cards}</div>
    </section>
    <section class="how-section" id="how">
      <div><span class="section-label">How this book works</span><h2>Incident. Bug. Fix. Second door. Exercise.</h2></div>
      <ol><li><span>01</span><strong>A real incident</strong><p>Told from the original disclosure or filing, with every fact gated by a claims file.</p></li><li><span>02</span><strong>The bug on Ledger</strong><p>A vulnerable and a fixed handler, and the few details that decide whether the fix is real.</p></li><li><span>03</span><strong>The second door</strong><p>Why the local fix is not enough, shown on a second route that reuses an earlier repair.</p></li><li><span>04</span><strong>An exercise that is a test</strong><p>A decoy, a near-identical pair with opposite outcomes, and answers that trace input, check and response.</p></li></ol>
    </section>
    <footer class="home-footer"><div><strong>${escapeHtml(book.name)}</strong><p>${escapeHtml(book.tagline)}</p></div><p>Ledger is a teaching example. No real company, product or person is implied. Method after Colin Domoney, <cite>Defending APIs</cite> (Packt, 2024).</p></footer>
  </main>`;
  return shell({ title: "Which check did this request skip?", description: book.description, body, pageClass: "home-view" });
}

fs.writeFileSync(path.join(out, "index.html"), homePage());
items.forEach((item, index) => fs.writeFileSync(path.join(out, "chapters", `${item.slug}.html`), chapterPage(item, index)));
for (const asset of ["styles.css", "app.js"]) fs.copyFileSync(path.join(here, asset), path.join(out, "assets", asset));
fs.writeFileSync(path.join(out, ".nojekyll"), "");
fs.writeFileSync(path.join(out, "404.html"), homePage());
const pendingTotal = items.reduce((n, item) => n + (item.source.match(/^\{\{excerpt:/gm) || []).length, 0);
console.log(`Built ${items.length} chapters in docs/ (${pendingTotal} excerpt placeholders pending)`);
