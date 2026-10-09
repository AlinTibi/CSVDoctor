export function escapeHTML(value) {
  return String(value ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
}
export function changedCells(before, after) {
  return after.map((row, r) =>
    row.cells.map(
      (cell, c) =>
        !before[r] ||
        c >= before[r].cells.length ||
        cell !== before[r].cells[c],
    ),
  );
}
export function findingGroups(counts, issues) {
  const groups = new Map(
    Object.keys(counts).map((code) => [
      code,
      {
        message:
          "Additional findings of this type exist beyond the preview location limit. See the count; review source data carefully.",
        severity: "warning",
        locations: [],
      },
    ]),
  );
  const described = new Set();
  for (const issue of issues) {
    const group = groups.get(issue.code);
    if (!group) continue;
    if (!described.has(issue.code)) {
      group.message = issue.message;
      group.severity = issue.severity;
      described.add(issue.code);
    }
    if (issue.row)
      group.locations.push(
        `R${issue.row}${issue.column ? ` C${issue.column}` : ""}`,
      );
  }
  return groups;
}
