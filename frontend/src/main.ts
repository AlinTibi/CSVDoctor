import "./style.css";
import {
  escapeHTML as esc,
  changedCells,
  findingGroups,
} from "./presentation.mjs";
import type {
  Document,
  Snapshot,
  ParseOptions,
  ExportOptions,
  Repairs,
  ReviewResult,
  Row,
} from "./types";

const icon = `<svg viewBox="0 0 48 48" aria-hidden="true"><rect x="6" y="6" width="36" height="36" rx="9" fill="#19283c"/><path d="M12 16h24M12 24h24M20 12v24M28 12v12" stroke="#80bdf4" stroke-width="2.4"/><circle cx="34" cy="34" r="10" fill="#e8b66e"/><path d="m29 34 3 3 6-7" fill="none" stroke="#172235" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"/></svg>`;
const root = document.querySelector<HTMLDivElement>("#app")!;
let source: Document | null = null,
  revision = "",
  review: ReviewResult | null = null,
  busy = false,
  view: "source" | "comparison" = "source";
root.innerHTML = `<header class="topbar"><div class="brand">${icon}<div><strong>CSV Doctor</strong><span>ALMARFELD <i>·</i> v1.0.0 candidate</span></div></div><span class="offline"><b></b> Offline workspace</span><button id="open" class="primary">Open file…</button></header>
<div class="workflow" aria-label="Workflow"><span class="step active">1 <b>Open file</b></span><i>→</i><span class="step" id="step-diagnose">2 <b>Diagnose</b></span><i>→</i><span class="step" id="step-review">3 <b>Review fixes</b></span><i>→</i><span class="step" id="step-export">4 <b>Export a copy</b></span><span class="guarantee">Your original stays unchanged</span></div>
<div id="notice" class="notice" role="status" aria-live="polite" hidden></div>
<main><section id="empty" class="empty"><div class="large-icon">${icon}</div><p class="eyebrow">DELIMITED TEXT, WITHOUT THE GUESSWORK</p><h1>Check your CSV before importing it.</h1><p class="lead">Find broken rows, quoting problems and spreadsheet risks.<br>Review every proposed change, then export a new copy.</p><button id="open-empty" class="primary">Open CSV, TSV or TXT…</button><p class="drop-hint">Or drop one file anywhere in this window</p><div class="empty-features"><span>Text stays text</span><span>No uploads or accounts</span><span>Originals never overwritten</span></div><p class="limits">v1 candidate · Files up to 16 MiB · 100,000 records · 256 columns</p></section>
<div id="loaded" hidden><section class="filebar"><div><span class="eyebrow">SOURCE FILE</span><h1 id="filename"></h1><p id="file-info"></p></div><span id="parse-status" class="badge"></span></section>
<div class="workspace"><aside><section class="panel"><div class="panel-heading"><h2>Source interpretation</h2><span class="tiny">Review detection</span></div><label for="input-encoding">Encoding</label><select id="input-encoding"><option value="auto">Detect automatically</option><option value="utf8">UTF-8</option><option value="utf8bom">UTF-8 BOM</option><option value="utf16le">UTF-16 little-endian</option><option value="utf16be">UTF-16 big-endian</option><option value="windows1252">Windows-1252</option></select><label for="input-delimiter">Delimiter</label><select id="input-delimiter"><option value="auto">Detect automatically</option><option value=",">Comma (,)</option><option value=";">Semicolon (;)</option><option value="\t">Tab</option><option value="|">Pipe (|)</option></select><label class="check"><input id="has-header" type="checkbox" checked> First record is a header</label><label for="quote-mode">Quote interpretation</label><select id="quote-mode"><option value="strict">Strict CSV quoting</option><option value="literal">Treat quotes as literal text</option></select><p id="literal-warning" class="caution" hidden>Explicit interpretation: quotes become ordinary characters and every line becomes a record. Multiline quoted cells may split. Compare carefully; this does not recover the author's intent.</p><button id="diagnose" class="secondary full">Diagnose with these choices</button></section>
<section class="panel"><div class="panel-heading"><h2>Export options</h2></div><label for="output-encoding">Encoding</label><select id="output-encoding"><option value="utf8bom">UTF-8 BOM — Excel compatibility</option><option value="utf8">UTF-8</option></select><label for="output-delimiter">Delimiter</label><select id="output-delimiter"><option value=",">Comma (,)</option><option value=";">Semicolon (;)</option><option value="\t">Tab</option><option value="|">Pipe (|)</option></select><label for="output-ending">Record line endings</label><select id="output-ending"><option value="\r\n">CRLF — Windows</option><option value="\n">LF</option></select><p class="help">Line breaks inside cells stay unchanged. A BOM helps encoding, not identifier or formula safety.</p></section></aside>
<div class="content"><section class="summary" id="summary"></section><section class="panel diagnosis"><div class="panel-heading"><h2>Diagnostics</h2><span class="tiny" id="issue-limit"></span></div><div id="issues"></div><details id="raw"><summary>Decoded source excerpt</summary><pre id="raw-text"></pre></details></section>
<section class="panel repairs"><div class="panel-heading"><h2>Choose repairs</h2><span class="tiny">Content changes are opt-in</span></div><div class="repair-grid"><label class="repair"><input id="repair-headers" type="checkbox"><span><strong>Fill empty / duplicate headers</strong><small>Names changed here are highlighted in the comparison.</small></span></label><label class="repair"><input id="repair-pad" type="checkbox"><span><strong>Pad short rows with empty fields</strong><small>Uses the widest row. Keeps every original field and record.</small></span></label><label class="repair"><input id="repair-formulas" type="checkbox"><span><strong>Prefix formula-risk cells with ' </strong><small>Changes cell text. May not survive resaving in Excel; not universal protection.</small></span></label><label class="repair"><input id="keep-uneven" type="checkbox"><span><strong>Keep uneven row lengths</strong><small>Explicitly leaves this finding unresolved; import may still fail.</small></span></label></div><p class="help">Identifiers are never converted to numbers or dates. For Excel, import identifier columns as <strong>Text</strong>; CSV quoting does not preserve spreadsheet types.</p><div class="review-action"><button id="review" class="secondary">Review proposed changes</button><span id="review-hint">Review is required before export.</span></div><div id="plan" hidden></div></section>
<section class="panel preview"><div class="panel-heading"><div><h2>Table preview</h2><span class="tiny" id="preview-limit"></span></div><div class="tabs" role="group" aria-label="Table view"><button id="source-tab" aria-pressed="true">Source</button><button id="compare-tab" aria-pressed="false" disabled>Before / after</button></div></div><div id="tables"></div></section></div></div>
<div class="exportbar"><span id="export-status">Review the proposed changes first.</span><button id="export" class="primary" disabled>Export repaired copy…</button></div></div></main><footer><span>No telemetry · No uploads · No automatic downloads</span><span id="status">Ready</span></footer>`;
const $ = <T extends HTMLElement = HTMLElement>(id: string) =>
  document.getElementById(id) as T;
const api = () => {
  const a = window.go?.main.App;
  if (!a)
    throw new Error("Desktop backend is unavailable. Run CSV Doctor.exe.");
  return a;
};
const labelDelimiter = (v: string) =>
  ({ ",": "Comma", ";": "Semicolon", "\t": "Tab", "|": "Pipe" })[v] ??
  "Unknown";
const labelEncoding = (v: string) =>
  ({
    utf8: "UTF-8",
    utf8bom: "UTF-8 BOM",
    utf16le: "UTF-16 LE",
    utf16be: "UTF-16 BE",
    windows1252: "Windows-1252",
  })[v] ?? "Unknown";
const parseOptions = (): ParseOptions => ({
  encoding: $<HTMLSelectElement>("input-encoding").value,
  delimiter: $<HTMLSelectElement>("input-delimiter").value,
  header: $<HTMLInputElement>("has-header").checked,
  quoteMode: $<HTMLSelectElement>("quote-mode").value,
});
const outputOptions = (): ExportOptions => ({
  encoding: $<HTMLSelectElement>("output-encoding").value,
  delimiter: $<HTMLSelectElement>("output-delimiter").value,
  lineEnding:
    $<HTMLSelectElement>("output-ending").selectedIndex === 0 ? "\r\n" : "\n",
});
const repairOptions = (): Repairs => ({
  headers: $<HTMLInputElement>("repair-headers").checked,
  padRows: $<HTMLInputElement>("repair-pad").checked,
  keepUneven: $<HTMLInputElement>("keep-uneven").checked,
  formulaProtection: $<HTMLInputElement>("repair-formulas").checked,
});
function notice(message: string, error = false) {
  $("notice").textContent = message;
  $("notice").hidden = !message;
  $("notice").classList.toggle("error", error);
}
async function task(operation: () => Promise<void>) {
  if (busy) return;
  busy = true;
  root.classList.add("busy");
  $("status").textContent = "Working locally…";
  try {
    await operation();
  } catch (e) {
    notice(String(e instanceof Error ? e.message : e), true);
  } finally {
    busy = false;
    root.classList.remove("busy");
    $("status").textContent = "Ready";
  }
}
function invalidate() {
  $("step-review").classList.remove("active");
  $("step-export").classList.remove("active");
  review = null;
  view = "source";
  $("plan").hidden = true;
  $<HTMLButtonElement>("export").disabled = true;
  $<HTMLButtonElement>("compare-tab").disabled = true;
  $("review-hint").textContent = "Options changed. Review again before export.";
  $("export-status").textContent = "Review the current options before export.";
  if (source) renderTable();
}
function accept(snapshot: Snapshot) {
  source = snapshot.document;
  revision = snapshot.revision;
  invalidate();
  $<HTMLButtonElement>("review").disabled = false;
  render();
}
async function open() {
  await task(async () => {
    notice("");
    const s = await api().OpenFile({
      encoding: "auto",
      delimiter: "auto",
      header: true,
      quoteMode: "strict",
    });
    if (s) {
      resetOptions();
      accept(s);
    }
  });
}
function resetOptions() {
  for (const id of ["input-encoding", "input-delimiter"])
    $<HTMLSelectElement>(id).value = "auto";
  $<HTMLSelectElement>("quote-mode").value = "strict";
  $<HTMLInputElement>("has-header").checked = true;
  for (const id of [
    "repair-headers",
    "repair-pad",
    "repair-formulas",
    "keep-uneven",
  ])
    $<HTMLInputElement>(id).checked = false;
  $("literal-warning").hidden = true;
}
function render() {
  $("empty").hidden = !!source;
  $("loaded").hidden = !source;
  if (!source) return;
  $("filename").textContent = source.name;
  $("file-info").textContent =
    `${(source.bytes / 1024).toFixed(1)} KiB · ${source.rowCount.toLocaleString()} records${source.parsed ? "" : " parsed before failure"} · ${source.columns} expected fields`;
  $("parse-status").textContent = !source.parsed
    ? "Parsing blocked"
    : !source.encodingConfirmed || !source.delimiterConfirmed
      ? "Confirm interpretation"
      : "Diagnosed";
  $("parse-status").classList.toggle(
    "blocked",
    !source.parsed || !source.encodingConfirmed || !source.delimiterConfirmed,
  );
  $("summary").innerHTML =
    `<div><span>Encoding</span><strong>${esc(labelEncoding(source.encoding))}</strong><small>${source.encodingConfirmed ? "Confirmed / detected" : "Needs your confirmation"}</small></div><div><span>Delimiter</span><strong>${esc(labelDelimiter(source.delimiter))}</strong><small>${source.delimiterConfirmed ? "Confirmed / detected" : "Ambiguous — choose explicitly"}</small></div><div><span>Line endings</span><strong>${esc(source.lineEnding || "Unknown")}</strong><small>Includes line breaks inside cells</small></div><div><span>Findings</span><strong>${Object.values(source.counts).reduce((a, b) => a + b, 0)}</strong><small>${source.counts["formula-risk"] || 0} formula · ${source.counts["identifier-risk"] || 0} identifier risks</small></div>`;
  const groups = findingGroups(source.counts, source.issues);
  $("issues").innerHTML = groups.size
    ? [...groups.entries()]
        .map(
          ([code, v]) =>
            `<details class="finding ${esc(v.severity)}" ${v.severity === "error" ? "open" : ""}><summary><span class="severity-dot"></span><strong>${esc(code.replaceAll("-", " "))}</strong><span class="finding-count">${source!.counts[code] || 0}</span></summary><p>${esc(v.message)}</p>${v.locations.length ? `<p class="locations">${esc(v.locations.slice(0, 12).join(" · "))}${v.locations.length > 12 ? " …" : ""}</p>` : ""}</details>`,
        )
        .join("")
    : '<div class="clean">No structural findings. Spreadsheet behavior still depends on import settings.</div>';
  $("issue-limit").textContent =
    source.issues.length >= 200 ? "First 200 finding locations" : "";
  $("raw-text").textContent = source.rawPreview;
  $("raw").hidden = source.parsed;
  $<HTMLInputElement>("repair-headers").disabled = !source.header;
  if (!source.header) $<HTMLInputElement>("repair-headers").checked = false;
  $("step-diagnose").classList.add("active");
  $("preview-limit").textContent =
    `First ${Math.min(source.rowCount, 100)} records · Long preview cells are truncated; exports use the full text.`;
  renderTable();
}
function table(rows: Row[], title: string, changes?: boolean[][]) {
  const count = Math.max(1, ...rows.map((r) => r.cells.length));
  const headings = Array.from(
    { length: count },
    (_, c) => `<th scope="col">${c + 1}</th>`,
  ).join("");
  return `<div class="table-side"><h3>${title}</h3><div class="table-scroll" tabindex="0" aria-label="${title} preview"><table><thead><tr><th scope="col" class="row-number">Record</th>${headings}</tr></thead><tbody>${rows.map((r, i) => `<tr ${source?.header && i === 0 ? 'class="header-row"' : ""}><th scope="row">${i + 1}<small>line ${r.line}</small></th>${Array.from({ length: count }, (_, c) => (c < r.cells.length ? `<td class="${changes?.[i]?.[c] ? "changed" : ""}"><span>${esc(r.cells[c]) || '<i class="empty-cell">empty</i>'}</span></td>` : '<td class="missing"><i>missing</i></td>')).join("")}</tr>`).join("")}</tbody></table>${rows.length ? "" : '<p class="no-rows">No complete records available.</p>'}</div></div>`;
}
function renderTable() {
  if (!source) return;
  const compare = view === "comparison" && review;
  $<HTMLButtonElement>("source-tab").setAttribute(
    "aria-pressed",
    String(!compare),
  );
  $<HTMLButtonElement>("compare-tab").setAttribute(
    "aria-pressed",
    String(!!compare),
  );
  $("tables").classList.toggle("comparison", !!compare);
  $("tables").innerHTML = compare
    ? table(review!.plan.before, "Before") +
      table(
        review!.plan.after,
        "After",
        changedCells(review!.plan.before, review!.plan.after),
      )
    : table(source.preview, "Source");
}
$("open").onclick = open;
$("open-empty").onclick = open;
$("diagnose").onclick = () =>
  task(async () => {
    notice("");
    accept(await api().Diagnose(parseOptions()));
  });
for (const id of [
  "input-encoding",
  "input-delimiter",
  "has-header",
  "quote-mode",
])
  $(id).onchange = () => {
    invalidate();
    $<HTMLButtonElement>("review").disabled = true;
    $("review-hint").textContent =
      "Diagnose with the new source choices first.";
    $("literal-warning").hidden =
      $<HTMLSelectElement>("quote-mode").value !== "literal";
  };
for (const id of [
  "output-encoding",
  "output-delimiter",
  "output-ending",
  "repair-headers",
  "repair-pad",
  "repair-formulas",
  "keep-uneven",
])
  $(id).onchange = () => {
    if (id === "repair-pad" && $<HTMLInputElement>(id).checked)
      $<HTMLInputElement>("keep-uneven").checked = false;
    if (id === "keep-uneven" && $<HTMLInputElement>(id).checked)
      $<HTMLInputElement>("repair-pad").checked = false;
    invalidate();
  };
$("review").onclick = () =>
  task(async () => {
    if (!source) return;
    notice("");
    review = await api().Review(revision, repairOptions(), outputOptions());
    view = "comparison";
    $("plan").hidden = false;
    $("plan").innerHTML =
      `<h3>Exactly what will be applied</h3><ul class="change-list">${review.plan.changes.map((c) => `<li><strong>${c.count} ${esc(c.kind.replaceAll("-", " "))}</strong><span>${esc(c.description)}</span></li>`).join("")}</ul>${review.plan.blockers.length ? `<div class="blockers"><strong>Export blocked</strong><ul>${review.plan.blockers.map((b) => `<li>${esc(b)}</li>`).join("")}</ul></div>` : '<p class="help">Export creates the selected file plus an adjacent JSON repair report. Existing files are refused.</p>'}`;
    $<HTMLButtonElement>("export").disabled = !review.plan.canExport;
    $<HTMLButtonElement>("compare-tab").disabled = false;
    $("review-hint").textContent = "Changed cells are highlighted in amber.";
    $("export-status").textContent = review.plan.canExport
      ? "Reviewed. Original rows and cell text are preserved except for your selected content changes."
      : "Resolve the review blockers before exporting.";
    $("step-review").classList.add("active");
    renderTable();
  });
$("export").onclick = () =>
  task(async () => {
    if (!review) return;
    const result = await api().Export(review.token);
    if (result.cancelled) {
      notice("Export cancelled. No files were created.");
      return;
    }
    notice(
      `Saved ${result.filename} and ${result.reportFilename}. Original unchanged. Remaining findings are recorded in the report.`,
    );
    $("step-export").classList.add("active");
  });
$("source-tab").onclick = () => {
  view = "source";
  renderTable();
};
$("compare-tab").onclick = () => {
  view = "comparison";
  renderTable();
};
window.runtime?.OnFileDrop((_x, _y, paths) => {
  if (busy) return;
  if (paths.length !== 1) {
    notice("Drop one CSV, TSV or TXT file at a time.", true);
    return;
  }
  void task(async () => {
    resetOptions();
    accept(await api().LoadFile(paths[0], parseOptions()));
    notice("");
  });
}, false);
void task(async () => {
  const s = await api().InitialDocument();
  if (s.document) accept(s);
});
