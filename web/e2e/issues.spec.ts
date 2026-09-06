import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";
import { BD_DATA_DIR, E2E_BASE_URL } from "../playwright.config.js";

// Drives the Issues tab against the Go preview server, which reads its issues
// through the fake bd on the daemon's PATH (see playwright.config webServer):
// the list and its state buttons, the query language, the labels page, and the
// detail page with its rendered fields and neighbourhood graph.
//
// The headless environment has no fonts, so text-only elements measure 0x0 and
// Playwright reports them hidden; assertions use toBeAttached and text content
// rather than visibility, and clicks land on elements that carry an icon or a
// drawn shape.

const fixtureDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), BD_DATA_DIR);

let sourceId: string;

test.beforeAll(async () => {
  // Connect RPC accepts plain JSON POSTs; sources are process-local and keyed
  // by the .beads path, so re-registering is idempotent daemon state.
  const resp = await fetch(`${E2E_BASE_URL}/ngicks.crabswarm.issues.v1.IssuesService/AddSource`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ dir: fixtureDir }),
  });
  expect(resp.ok).toBeTruthy();
  const body = (await resp.json()) as { source: { id: string; beadsPath: string } };
  // Pins the run to the fake bd: a daemon started without it on PATH resolves
  // this directory to the repository's own beads database instead, and every
  // assertion below would fail for a reason that has nothing to do with the SPA.
  expect(body.source.beadsPath).toBe("/fake/crabswarm/.beads");
  sourceId = body.source.id;
});

function listUrl(query = ""): string {
  return `/issues/${encodeURIComponent(sourceId)}${query}`;
}

function issueUrl(issueId: string, query = ""): string {
  return `/issues/${encodeURIComponent(sourceId)}/${issueId}${query}`;
}

test("the source picker lists the registered source and opens it", async ({ page }) => {
  await page.goto("/issues");
  const entry = page.getByRole("link", { name: "crabswarm" }).first();
  await expect(entry).toHaveAttribute("href", listUrl());
});

test("the list shows the open issues, with counts on the state buttons", async ({ page }) => {
  await page.goto(listUrl());

  const rows = page.getByTestId("issue-list").locator("tbody tr");
  await expect(rows).toHaveCount(2);
  await expect(rows.first()).toContainText("crabswarm-no2");
  await expect(rows.nth(1)).toContainText("crabswarm-jp7");

  // "how many if I click": the rest of the query evaluated with that state.
  await expect(page.getByTestId("state-open")).toContainText("2 Open");
  await expect(page.getByTestId("state-closed")).toContainText("1 Closed");
  await expect(page.getByTestId("state-plans")).toContainText("1 Plans");
});

test("the Closed button swaps the state token and lists the closed issue", async ({ page }) => {
  await page.goto(listUrl());
  await page.getByTestId("state-closed").click();

  await expect(page).toHaveURL(`${E2E_BASE_URL}${listUrl("?q=is:closed")}`);
  const rows = page.getByTestId("issue-list").locator("tbody tr");
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText("crabswarm-125");
});

test("a label: query narrows the list", async ({ page }) => {
  await page.goto(listUrl("?q=is:open label:tui"));

  const rows = page.getByTestId("issue-list").locator("tbody tr");
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText("crabswarm-jp7");
  await expect(page.getByTestId("query-input")).toHaveValue("is:open label:tui");
  await expect(page.getByTestId("query-status")).toContainText("1 issue");
});

test("a label whose name has a space stays one token through the bar", async ({ page }) => {
  // The labels page writes the token the bar reads back: a value with
  // whitespace is quoted, and `q` is form-encoded, so the quotes are %22.
  await page.goto(`/issues/${encodeURIComponent(sourceId)}/labels`);
  await page.getByTestId("labels-search").fill("needs");
  const quoted = listUrl("?q=is:open+label:%22needs+review%22");
  await expect(page.getByTestId("label-name")).toHaveAttribute("href", quoted);

  await page.goto(quoted);
  const rows = page.getByTestId("issue-list").locator("tbody tr");
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText("crabswarm-jp7");
  await expect(page.getByTestId("query-input")).toHaveValue('is:open label:"needs review"');

  // The state buttons rewrite the query token by token, so the quoted value
  // has to come back whole rather than split at its space.
  await page.getByTestId("state-closed").click();
  await expect(page.getByTestId("query-input")).toHaveValue('label:"needs review" is:closed');
  await expect(page.getByTestId("issue-list")).toContainText("No issue matches the query.");
});

test("a half-typed quoted value is the token the suggestions answer for", async ({ page }) => {
  await page.goto(listUrl());
  // The space inside an open quote belongs to the `label:` token, so the bar
  // answers for that value — no label matches, and none of the qualifier
  // names it would offer for a token that had just begun show up either.
  await page.getByTestId("query-input").fill('is:open label:"needs ');

  await expect(page.getByText("No suggestion for this token")).toBeAttached();
  await expect(page.getByTestId("query-suggestion")).toHaveCount(0);
});

test("an unknown qualifier is named under the bar and matches nothing", async ({ page }) => {
  await page.goto(listUrl("?q=milestone:v1"));

  await expect(page.getByTestId("query-unknown")).toContainText("milestone");
  await expect(page.getByTestId("issue-list")).toContainText("No issue matches the query.");
});

test("the labels page splits active from archived", async ({ page }) => {
  await page.goto(listUrl());
  await page.getByTestId("labels-link").click();
  await expect(page).toHaveURL(`${E2E_BASE_URL}/issues/${encodeURIComponent(sourceId)}/labels`);

  // Six labels are carried by an open issue; `retired` only by the closed one.
  await expect(page.getByTestId("labels-active")).toContainText("Active 6");
  await expect(page.getByTestId("labels-archived")).toContainText("Archived 1");
  const names = page.getByTestId("label-name");
  await expect(names).toHaveCount(6);
  await expect(page.getByTestId("labels-table")).not.toContainText("retired");

  await page.getByTestId("labels-archived").click();
  await expect(page.getByTestId("label-name")).toHaveCount(1);
  await expect(page.getByTestId("labels-table")).toContainText("retired");

  // A label links back to the list as the query the bar would hold. The `q`
  // is form-encoded, so its separators are `+` and its colons stay readable.
  await expect(page.getByTestId("label-name").first()).toHaveAttribute(
    "href",
    listUrl("?q=is:open+label:retired"),
  );
});

test("the labels filter narrows by name", async ({ page }) => {
  await page.goto(`/issues/${encodeURIComponent(sourceId)}/labels`);
  await page.getByTestId("labels-search").fill("pro");

  await expect(page.getByTestId("label-name")).toHaveCount(1);
  await expect(page.getByTestId("labels-table")).toContainText("proto");
});

test("the detail page renders the fields, the outline and the diagram", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-no2"));

  await expect(page.getByRole("heading", { name: "Plans in beads", level: 1 })).toBeAttached();
  await expect(page.getByTestId("metadata")).toContainText("idea_gate_passed=yes");

  // Every rendered section the fixture carries, in page order.
  for (const title of ["Description", "Design", "Acceptance criteria", "Notes"]) {
    await expect(page.getByRole("heading", { name: title, level: 2 })).toBeAttached();
  }
  await expect(page.getByRole("heading", { name: "Dependencies (2)", level: 2 })).toBeAttached();
  await expect(page.getByRole("heading", { name: "Comments (1)", level: 2 })).toBeAttached();

  // The ```mermaid fence in the description is drawn client-side.
  await expect(page.locator("#section-description pre.mermaid svg")).toBeAttached();

  // The outline lists every section, with the description's own headings under it.
  const toc = page.getByTestId("toc");
  await expect(toc).toContainText("Description");
  await expect(toc).toContainText("Shape");
  await expect(toc).toContainText("Neighbourhood");

  // The comment's `Decision:` prefix becomes a badge.
  await expect(page.getByTestId("comments")).toContainText("Decision");
});

test("the neighbourhood draws the issue's edges and a node opens its issue", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-no2"));

  const graph = page.getByTestId("local-graph");
  await expect(graph).toContainText("3 issues, 3 edges");
  const canvas = page.getByTestId("graph-canvas");
  await expect(canvas.locator("svg")).toBeAttached();
  await expect(canvas.locator("g.node")).toHaveCount(3);

  // Node ids carry their declaration index; n0 is the issue itself and n1 the
  // first neighbour, which is the `blocks` dependency.
  await canvas.locator('g.node[id*="flowchart-n1-"]').click();
  await expect(page).toHaveURL(`${E2E_BASE_URL}${issueUrl("crabswarm-jp7")}`);
  await expect(page.getByRole("heading", { level: 1 })).toContainText("WatchRoom upgrade path");
});

test("the query is carried onto the detail page and back", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-no2", "?q=is:open label:chat"));

  const back = page.getByTestId("detail-bar").getByRole("link", { name: "back to the list" });
  await expect(back).toHaveAttribute("href", listUrl("?q=is:open+label:chat"));
});

test("the Jump to menu reaches every section of the page", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-no2"));
  await page.getByTestId("jump-menu-trigger").click();

  const content = page.getByTestId("jump-menu-content");
  await expect(content.getByTestId("jump-to-description")).toBeAttached();
  await expect(content.getByTestId("jump-to-neighbourhood")).toBeAttached();
  await expect(content.getByTestId("jump-to-comments")).toBeAttached();
});

test("a closed issue shows its close reason", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-125"));

  await expect(page.getByRole("heading", { name: "Close reason", level: 2 })).toBeAttached();
  await expect(page.locator("#section-close")).toContainText("crabswarm chat admin tui");
});

test("an issue at the receiving end of its edges still shows them", async ({ page }) => {
  // Nothing in crabswarm-125's own record names an edge: bd reports only the
  // edges an issue carries, and both of this one's are carried by the issues
  // pointing at it. The page reads them off the source's whole edge list.
  await page.goto(issueUrl("crabswarm-125"));

  const deps = page.locator("#section-dependencies");
  await expect(page.getByRole("heading", { name: "Dependencies (2)", level: 2 })).toBeAttached();
  // Worded from this issue's side: it is what the other was discovered from.
  await expect(deps).toContainText("discovered");
  await expect(deps).toContainText("crabswarm-no2");
  await expect(deps).toContainText("related to");
  await expect(deps).toContainText("crabswarm-jp7");
  // The title comes from the source's listing, not from the edge.
  await expect(deps).toContainText("Plans in beads");

  const canvas = page.getByTestId("graph-canvas");
  await expect(page.getByTestId("local-graph")).toContainText("3 issues, 3 edges");
  await expect(canvas.locator("g.node")).toHaveCount(3);
});

test("an unknown issue id says so rather than blanking the page", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-zzz"));
  await expect(page.getByText("No issue crabswarm-zzz in this source.")).toBeAttached();
});
