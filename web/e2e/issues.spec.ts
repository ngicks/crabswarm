import { readFileSync } from "node:fs";
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

/** One record of the recorded listing, as far as the expectations below read
 *  it. The daemon derives the whole board from these records, so the counts
 *  come off the fixture rather than being restated as literals a re-recording
 *  would silently invalidate. */
interface RecordedIssue {
  id: string;
  status: string;
  parent?: string;
  labels?: string[];
  updated_at: string;
}

const recorded = JSON.parse(readFileSync(path.join(fixtureDir, "list.json"), "utf8")) as RecordedIssue[];

/** `is:open` is every status but closed, which is bd's own default listing. */
const openIssues = recorded.filter((r) => r.status !== "closed");
const closedIssues = recorded.filter((r) => r.status === "closed");

const labelsOf = (r: RecordedIssue) => r.labels ?? [];
const carrying = (rows: RecordedIssue[], label: string) => rows.filter((r) => labelsOf(r).includes(label));
const distinctLabels = (rows: RecordedIssue[]) => new Set(rows.flatMap(labelsOf));

/** The id the list puts first: newest updated, ties broken by id, the order
 *  the daemon sorts in and the page keeps. bd stamps RFC 3339, so the strings
 *  compare as the times do. */
function firstRow(rows: RecordedIssue[]): string {
  return [...rows].sort((a, b) => b.updated_at.localeCompare(a.updated_at) || a.id.localeCompare(b.id))[0].id;
}

/** A label is active while an issue that is not closed carries it, and
 *  archived once only closed ones do. */
const activeLabels = distinctLabels(openIssues);
const archivedLabels = [...distinctLabels(closedIssues)].filter((l) => !activeLabels.has(l)).sort();

/** The issue the detail tests open: an epic carrying every text field, a
 *  mermaid diagram, metadata, children, comments, and one dependency edge that
 *  only the issue at the other end carries. */
const EPIC = "crabswarm-3hp";
/** A closed child of that epic: a close reason, and dependency edges running
 *  both ways. */
const STEP = "crabswarm-3hp.2";
const epicChildren = recorded.filter((r) => r.parent === EPIC);

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
  await expect(rows).toHaveCount(openIssues.length);
  await expect(rows.first()).toContainText(firstRow(openIssues));

  // "how many if I click": the rest of the query evaluated with that state.
  await expect(page.getByTestId("state-open")).toContainText(`${openIssues.length} Open`);
  await expect(page.getByTestId("state-closed")).toContainText(`${closedIssues.length} Closed`);
  await expect(page.getByTestId("state-plans")).toContainText(`${carrying(openIssues, "plan").length} Plans`);
});

test("the Closed button swaps the state token and lists the closed issues", async ({ page }) => {
  await page.goto(listUrl());
  await page.getByTestId("state-closed").click();

  await expect(page).toHaveURL(`${E2E_BASE_URL}${listUrl("?q=is:closed")}`);
  const rows = page.getByTestId("issue-list").locator("tbody tr");
  await expect(rows).toHaveCount(closedIssues.length);
  await expect(rows.first()).toContainText(firstRow(closedIssues));
});

test("a label: query narrows the list", async ({ page }) => {
  const tui = carrying(openIssues, "tui");
  await page.goto(listUrl("?q=is:open label:tui"));

  const rows = page.getByTestId("issue-list").locator("tbody tr");
  await expect(rows).toHaveCount(tui.length);
  await expect(rows.first()).toContainText(firstRow(tui));
  await expect(page.getByTestId("query-input")).toHaveValue("is:open label:tui");
  await expect(page.getByTestId("query-status")).toContainText(`${tui.length} issues`);
});

test("a label whose name has a space stays one token through the bar", async ({ page }) => {
  // No recorded label carries whitespace, so the value below matches nothing;
  // what the test is about is the token surviving the round trip. `q` is
  // form-encoded, so the quotes arrive as %22.
  await page.goto(listUrl("?q=is:open+label:%22needs+review%22"));

  await expect(page.getByTestId("query-input")).toHaveValue('is:open label:"needs review"');
  await expect(page.getByTestId("issue-list")).toContainText("No issue matches the query.");

  // The state buttons rewrite the query token by token, so the quoted value
  // has to come back whole rather than split at its space.
  await page.getByTestId("state-closed").click();
  await expect(page.getByTestId("query-input")).toHaveValue('label:"needs review" is:closed');
  await expect(page.getByTestId("issue-list")).toContainText("No issue matches the query.");
});

test("a half-typed quoted value is the token the suggestions answer for", async ({ page }) => {
  await page.goto(listUrl());
  // The space inside an open quote belongs to the `label:` token, so the bar
  // answers for that value. The opening quote is part of the value, which is
  // why the recorded label beginning with `needs` is not offered either, and
  // none of the qualifier names it would offer for a token that had just begun
  // show up.
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

  await expect(page.getByTestId("labels-active")).toContainText(`Active ${activeLabels.size}`);
  await expect(page.getByTestId("labels-archived")).toContainText(`Archived ${archivedLabels.length}`);
  const names = page.getByTestId("label-name");
  await expect(names).toHaveCount(activeLabels.size);
  await expect(page.getByTestId("labels-table")).not.toContainText(archivedLabels[0]);

  await page.getByTestId("labels-archived").click();
  await expect(page.getByTestId("label-name")).toHaveCount(archivedLabels.length);
  await expect(page.getByTestId("labels-table")).toContainText(archivedLabels[0]);

  // A label links back to the list as the query the bar would hold. The `q`
  // is form-encoded, so its separators are `+` and its colons stay readable.
  await expect(page.getByTestId("label-name").first()).toHaveAttribute(
    "href",
    listUrl(`?q=is:open+label:${archivedLabels[0]}`),
  );
});

test("the labels filter narrows by name", async ({ page }) => {
  await page.goto(`/issues/${encodeURIComponent(sourceId)}/labels`);
  await page.getByTestId("labels-search").fill("pro");

  await expect(page.getByTestId("label-name")).toHaveCount(1);
  await expect(page.getByTestId("labels-table")).toContainText("proto");
});

test("the detail page renders the fields, the outline and the diagram", async ({ page }) => {
  await page.goto(issueUrl(EPIC));

  await expect(
    page.getByRole("heading", { name: "Issues board reads from one bd queue", level: 1 }),
  ).toBeAttached();
  await expect(page.getByTestId("metadata")).toContainText("idea_gate_passed=2026-09-07");

  // Every rendered section the fixture carries, in page order.
  for (const title of ["Description", "Design", "Acceptance criteria", "Notes"]) {
    await expect(page.getByRole("heading", { name: title, level: 2 })).toBeAttached();
  }
  await expect(
    page.getByRole("heading", { name: `Children (${epicChildren.length})`, level: 2 }),
  ).toBeAttached();
  await expect(page.locator("#section-children").locator("tbody tr")).toHaveCount(epicChildren.length);
  await expect(page.getByRole("heading", { name: "Comments (18)", level: 2 })).toBeAttached();

  // The ```mermaid fence in the description is drawn client-side.
  await expect(page.locator("#section-description pre.mermaid svg")).toBeAttached();

  // The outline lists every section, with the description's own headings under it.
  const toc = page.getByTestId("toc");
  await expect(toc).toContainText("Description");
  await expect(toc).toContainText("Use cases");
  await expect(toc).toContainText("Neighbourhood");

  // The comment's `Decision:` prefix becomes a badge.
  await expect(page.getByTestId("comments")).toContainText("Decision");
});

test("an issue at the receiving end of its edges still shows them", async ({ page }) => {
  // Nothing in this issue's own record names an edge: bd reports only the
  // edges an issue carries, and this one's is carried by the issue pointing at
  // it. The page reads it off the source's whole edge list.
  await page.goto(issueUrl(EPIC));

  const deps = page.locator("#section-dependencies");
  await expect(page.getByRole("heading", { name: "Dependencies (1)", level: 2 })).toBeAttached();
  // Worded from this issue's side: it is what the other was discovered from.
  await expect(deps).toContainText("discovered");
  await expect(deps).toContainText("crabswarm-46o");
  // The title comes from the source's listing, not from the edge.
  await expect(deps).toContainText("Align @playwright/test with the nix-provided chromium 1234");
});

test("the neighbourhood draws the issue and everything one edge away", async ({ page }) => {
  await page.goto(issueUrl(EPIC));

  // The issue, its seven children and the issue discovered from it; the edges
  // are the seven parent links, the seven blocks among the children, and that
  // one discovery.
  const graph = page.getByTestId("local-graph");
  await expect(graph).toContainText("9 issues, 15 edges");
  const canvas = page.getByTestId("graph-canvas");
  await expect(canvas.locator("svg")).toBeAttached();
  await expect(canvas.locator("g.node")).toHaveCount(9);
});

test("the query is carried onto the detail page and back", async ({ page }) => {
  await page.goto(issueUrl(EPIC, "?q=is:open label:chat"));

  const back = page.getByTestId("detail-bar").getByRole("link", { name: "back to the list" });
  await expect(back).toHaveAttribute("href", listUrl("?q=is:open+label:chat"));
});

test("the Jump to menu reaches every section of the page", async ({ page }) => {
  await page.goto(issueUrl(EPIC));
  await page.getByTestId("jump-menu-trigger").click();

  const content = page.getByTestId("jump-menu-content");
  await expect(content.getByTestId("jump-to-description")).toBeAttached();
  await expect(content.getByTestId("jump-to-neighbourhood")).toBeAttached();
  await expect(content.getByTestId("jump-to-comments")).toBeAttached();
});

test("a closed issue shows its close reason", async ({ page }) => {
  await page.goto(issueUrl(STEP));

  await expect(page.getByRole("heading", { name: "Close reason", level: 2 })).toBeAttached();
  await expect(page.locator("#section-close")).toContainText("8 concurrent identical List calls record one invocation");
});

test("the dependency table words each edge from this issue's side", async ({ page }) => {
  await page.goto(issueUrl(STEP));

  const deps = page.locator("#section-dependencies");
  await expect(page.getByRole("heading", { name: "Dependencies (3)", level: 2 })).toBeAttached();
  // The edge this issue carries, and the two carried by the issues pointing at
  // it: the same relation reads the other way round from each side.
  const rows = deps.locator("tbody tr");
  await expect(rows.filter({ hasText: "crabswarm-3hp.1" })).toContainText("depends on");
  // Both rows carry a `blocks` badge, so the wording is what tells the two
  // ends of that relation apart.
  await expect(rows.filter({ hasText: "crabswarm-3hp.3" })).toContainText("blocks");
  await expect(rows.filter({ hasText: "crabswarm-3hp.3" })).not.toContainText("depends on");
  await expect(rows.filter({ hasText: "crabswarm-0m5" })).toContainText("discovered");
  // The titles come from the source's listing, not from the edges.
  await expect(deps).toContainText("Poller.Refresh: one shared listing per source");
});

test("a node of the neighbourhood opens its issue", async ({ page }) => {
  await page.goto(issueUrl(STEP));

  // This issue, its parent, and the three issues its edges reach.
  const graph = page.getByTestId("local-graph");
  await expect(graph).toContainText("5 issues, 6 edges");
  const canvas = page.getByTestId("graph-canvas");
  await expect(canvas.locator("g.node")).toHaveCount(5);

  // Node ids carry their declaration index; n0 is the issue itself and n1 the
  // first neighbour, which is its parent.
  await canvas.locator('g.node[id*="flowchart-n1-"]').click();
  await expect(page).toHaveURL(`${E2E_BASE_URL}${issueUrl(EPIC)}`);
  await expect(page.getByRole("heading", { level: 1 })).toContainText("Issues board reads from one bd queue");
});

test("an unknown issue id says so rather than blanking the page", async ({ page }) => {
  await page.goto(issueUrl("crabswarm-zzz"));
  await expect(page.getByText("No issue crabswarm-zzz in this source.")).toBeAttached();
});
