package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"time"
)

const maxPreviewItems = 8
const previewWindow = 30 * 24 * time.Hour

// languageLabels defines the fixed display order (fr before de) for the
// homepage — deliberately a slice, not a range over supportedLanguages,
// since map iteration order is non-deterministic.
var languageLabels = []struct{ Code, Label string }{
	{"fr", "French"},
	{"de", "German"},
}

type indexCategory struct {
	Name        string
	FeedPath    string
	PreviewPath string
}

type indexLanguage struct {
	Code       string
	Label      string
	Categories []indexCategory
}

func buildIndexPageData() []indexLanguage {
	langs := make([]indexLanguage, 0, len(languageLabels))
	for _, l := range languageLabels {
		cats := allCategories(l.Code)
		il := indexLanguage{Code: l.Code, Label: l.Label, Categories: make([]indexCategory, 0, len(cats))}
		for _, cat := range cats {
			slug := slugify(cat)
			il.Categories = append(il.Categories, indexCategory{
				Name:        cat,
				FeedPath:    "/feeds/" + l.Code + "/" + slug + ".xml",
				PreviewPath: "/api/entries/" + l.Code + "/" + slug,
			})
		}
		langs = append(langs, il)
	}
	return langs
}

var indexTemplate = template.Must(template.New("index").Parse(indexTemplateSrc))

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	var buf bytes.Buffer
	if err := indexTemplate.Execute(&buf, buildIndexPageData()); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

type previewEntry struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	PubDate string `json:"pubDate"`
}

// serveCategoryPreview returns the latest entries from the last 30 days for
// a (lang, category) pair as JSON, used by the homepage modal to preview a
// subfeed without leaving the page.
func serveCategoryPreview(w http.ResponseWriter, db *sql.DB, lang, category string) {
	since := time.Now().Add(-previewWindow)
	items, err := entriesByCategorySince(db, lang, category, since, maxPreviewItems)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// entriesByCategory returns a nil slice when there are no rows; build
	// with make(..., 0, ...) so an empty result marshals to "[]", not "null".
	out := make([]previewEntry, 0, len(items))
	for _, it := range items {
		out = append(out, previewEntry{Title: it.Title, Link: it.Link, PubDate: it.PubDate.Format(time.RFC3339)})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(out)
}

const indexTemplateSrc = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ARTE — RSS feeds by category</title>
<style>
:root {
  --accent: #e1000f;
  --bg: #f7f6f4;
  --card-bg: #ffffff;
  --text: #1b1b1b;
  --muted: #6b6b6b;
  --border: #e3e1de;
  --radius: 10px;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.4;
}
header {
  padding: 2.5rem 1.5rem 1.5rem;
  text-align: center;
}
header h1 {
  margin: 0 0 .4rem;
  font-size: 1.9rem;
}
header p {
  margin: 0;
  color: var(--muted);
}
main {
  max-width: 1080px;
  margin: 0 auto;
  padding: 0 1.5rem 3rem;
}
section.lang-section {
  margin-top: 2.2rem;
}
section.lang-section h2 {
  border-bottom: 2px solid var(--accent);
  display: inline-block;
  padding-bottom: .2rem;
  font-size: 1.3rem;
}
.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: .6rem;
  margin-top: 1rem;
}
@media (max-width: 640px) {
  .grid {
    grid-template-columns: 1fr;
  }
}
.card {
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: .55rem .7rem;
  display: flex;
  align-items: center;
  gap: .6rem;
  transition: box-shadow .15s ease, transform .15s ease;
}
.card:hover {
  box-shadow: 0 4px 14px rgba(0, 0, 0, .07);
  transform: translateY(-1px);
}
.card h3 {
  margin: 0;
  font-size: .92rem;
  font-weight: 600;
  flex: 1 1 auto;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.actions {
  display: flex;
  flex-wrap: nowrap;
  flex-shrink: 0;
  gap: .35rem;
}
.btn {
  appearance: none;
  border: 1px solid var(--border);
  background: #fff;
  color: var(--text);
  font-size: .8rem;
  padding: .35rem .6rem;
  border-radius: 7px;
  cursor: pointer;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: .3rem;
  white-space: nowrap;
}
.btn:hover { border-color: var(--accent); color: var(--accent); }
.btn.feed-link {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
.btn.feed-link:hover { opacity: .9; color: #fff; }
.btn.icon-btn {
  padding: .35rem;
  width: 2rem;
  height: 2rem;
  justify-content: center;
}
.btn.icon-btn .check-icon { display: none; }
.btn.icon-btn.copied { border-color: #1a8a3d; color: #1a8a3d; }
.btn.icon-btn.copied .copy-icon { display: none; }
.btn.icon-btn.copied .check-icon { display: block; }
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}
:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
.modal {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  z-index: 100;
}
.modal[hidden] { display: none; }
.modal-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(20, 18, 17, .55);
}
.modal-content {
  position: relative;
  background: var(--card-bg);
  border-radius: var(--radius);
  max-width: 560px;
  width: 100%;
  max-height: 80vh;
  overflow: auto;
  padding: 1.4rem 1.6rem;
}
.modal-close {
  position: absolute;
  top: .6rem;
  right: .7rem;
  border: none;
  background: none;
  font-size: 1.4rem;
  line-height: 1;
  cursor: pointer;
  color: var(--muted);
}
.modal-close:hover { color: var(--accent); }
#modal-title { margin: 0 1.6rem .8rem 0; font-size: 1.15rem; }
.modal-status { color: var(--muted); }
.preview-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: .7rem;
}
.preview-list li {
  border-bottom: 1px solid var(--border);
  padding-bottom: .6rem;
}
.preview-list a {
  color: var(--text);
  font-weight: 600;
  text-decoration: none;
}
.preview-list a:hover { color: var(--accent); text-decoration: underline; }
.preview-list time {
  display: block;
  color: var(--muted);
  font-size: .82rem;
  margin-top: .15rem;
}
</style>
</head>
<body>
<header>
  <h1>ARTE — RSS feeds by category</h1>
  <p>One RSS feed per category and language, regenerated from ARTE's daily programme.</p>
</header>
<main>
{{range .}}
  <section class="lang-section" aria-labelledby="lang-{{.Code}}">
    <h2 id="lang-{{.Code}}">{{.Label}}</h2>
    <div class="grid">
    {{range .Categories}}
      <article class="card">
        <h3 title="{{.Name}}">{{.Name}}</h3>
        <div class="actions">
          <a class="btn feed-link" href="{{.FeedPath}}">RSS feed</a>
          <button type="button" class="btn icon-btn copy-btn" aria-label="Copy link" title="Copy link">
            <svg class="copy-icon" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="12" height="12" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
            <svg class="check-icon" viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12"></polyline></svg>
          </button>
          <button type="button" class="btn preview-btn" data-preview="{{.PreviewPath}}" data-name="{{.Name}}">Preview</button>
        </div>
      </article>
    {{end}}
    </div>
  </section>
{{end}}
</main>

<div id="copy-status" class="sr-only" aria-live="polite"></div>

<div id="preview-modal" class="modal" hidden aria-hidden="true" role="dialog" aria-modal="true" aria-labelledby="modal-title">
  <div class="modal-backdrop" data-close></div>
  <div class="modal-content">
    <button type="button" class="modal-close" data-close aria-label="Close">&times;</button>
    <h2 id="modal-title"></h2>
    <div id="modal-body"></div>
  </div>
</div>

<script>
document.querySelectorAll(".copy-btn").forEach(function (btn) {
  btn.addEventListener("click", async function () {
    var url = btn.closest(".card").querySelector(".feed-link").href;
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(url);
      } else {
        fallbackCopy(url);
      }
    } catch (e) {
      fallbackCopy(url);
    }
    flashCopied(btn);
  });
});

function fallbackCopy(text) {
  var ta = document.createElement("textarea");
  ta.value = text;
  ta.style.position = "fixed";
  ta.style.opacity = "0";
  document.body.appendChild(ta);
  ta.select();
  try { document.execCommand("copy"); } catch (e) {}
  ta.remove();
}

function flashCopied(btn) {
  btn.classList.add("copied");
  btn.setAttribute("aria-label", "Copied!");
  btn.setAttribute("title", "Copied!");
  document.getElementById("copy-status").textContent = "Link copied to clipboard";
  setTimeout(function () {
    btn.classList.remove("copied");
    btn.setAttribute("aria-label", "Copy link");
    btn.setAttribute("title", "Copy link");
  }, 1500);
}

var modal = document.getElementById("preview-modal");
var lastFocused = null;

document.querySelectorAll(".preview-btn").forEach(function (btn) {
  btn.addEventListener("click", function () {
    openPreview(btn.dataset.preview, btn.dataset.name, btn);
  });
});

async function openPreview(url, name, trigger) {
  lastFocused = trigger;
  document.getElementById("modal-title").textContent = name;
  var body = document.getElementById("modal-body");
  body.innerHTML = "";
  var loading = document.createElement("p");
  loading.className = "modal-status";
  loading.textContent = "Loading…";
  body.appendChild(loading);

  modal.hidden = false;
  modal.setAttribute("aria-hidden", "false");
  modal.querySelector(".modal-close").focus();

  try {
    var res = await fetch(url, { headers: { Accept: "application/json" } });
    if (!res.ok) throw new Error(String(res.status));
    renderEntries(await res.json(), body);
  } catch (e) {
    body.innerHTML = "";
    var err = document.createElement("p");
    err.className = "modal-status";
    err.textContent = "Failed to load entries.";
    body.appendChild(err);
  }
}

function renderEntries(items, body) {
  body.innerHTML = "";
  if (!items.length) {
    var empty = document.createElement("p");
    empty.className = "modal-status";
    empty.textContent = "No entries yet.";
    body.appendChild(empty);
    return;
  }
  var ul = document.createElement("ul");
  ul.className = "preview-list";
  items.forEach(function (it) {
    var li = document.createElement("li");
    var a = document.createElement("a");
    if (/^https?:\/\//i.test(it.link)) {
      a.href = it.link;
    }
    a.textContent = it.title;
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    var time = document.createElement("time");
    var d = new Date(it.pubDate);
    time.textContent = isNaN(d) ? it.pubDate : d.toLocaleString();
    li.appendChild(a);
    li.appendChild(time);
    ul.appendChild(li);
  });
  body.appendChild(ul);
}

function closeModal() {
  modal.hidden = true;
  modal.setAttribute("aria-hidden", "true");
  if (lastFocused) lastFocused.focus();
}

modal.addEventListener("click", function (e) {
  if (e.target.hasAttribute("data-close")) closeModal();
});
document.addEventListener("keydown", function (e) {
  if (e.key === "Escape" && !modal.hidden) closeModal();
});
</script>
</body>
</html>
`
