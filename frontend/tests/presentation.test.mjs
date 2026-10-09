import { test } from "node:test";
import assert from "node:assert/strict";
import {
  escapeHTML,
  changedCells,
  findingGroups,
} from "../src/presentation.mjs";
test("file contents cannot become HTML or execute scripts", () => {
  assert.equal(
    escapeHTML("<script>\"&'</script>"),
    "&lt;script&gt;&quot;&amp;&#39;&lt;/script&gt;",
  );
});
test("comparison highlights changed and padded cells, not preserved identifiers", () => {
  const b = [{ cells: ["00123", "12345678901234567890", "=1+2"] }],
    a = [{ cells: ["00123", "12345678901234567890", "'=1+2", ""] }];
  assert.deepEqual(changedCells(b, a), [[false, false, true, true]]);
});

test("finding categories remain visible after the 200-location preview limit", () => {
  const groups = findingGroups({ "identifier-risk": 200, "empty-header": 1 }, [
    {
      code: "identifier-risk",
      severity: "warning",
      message: "Import as Text",
      row: 2,
      column: 1,
    },
  ]);
  assert.equal(groups.size, 2);
  assert.ok(groups.get("empty-header").message.includes("limit"));
  assert.deepEqual(groups.get("identifier-risk").locations, ["R2 C1"]);
});
