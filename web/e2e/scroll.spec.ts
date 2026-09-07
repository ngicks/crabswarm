import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test, type Page } from "@playwright/test";
import { BD_DATA_DIR, E2E_BASE_URL } from "../playwright.config.js";

// Holds the app to one scrollbar. Every surface scrolls in a column of its own
// (components/Layout sizes itself to the viewport, index.css clips the
// document), so however tall the document under it grows, the window itself
// must never gain a second bar beside that column's.
//
// The headless environment has no fonts, so text-only elements measure 0x0 and
// the fixtures alone never fill the frame: each case drops a block taller than
// any viewport into the column first, which is the document a badly built shell
// would push the window past. Assertions read geometry, not visibility.

const here = path.dirname(fileURLToPath(import.meta.url));
/** Recorded `bd` output, served by the fake bd on the daemon's PATH. */
const bdFixtureDir = path.resolve(here, BD_DATA_DIR);
/** Real files for the Roots surface: the mermaid document the other specs use. */
const rootFixtureDir = path.join(here, "fixtures");

/** The epic whose detail page carries the neighbourhood graph. Drawing it makes
 *  mermaid hang a tooltip off the body, outside the shell — the node that used
 *  to stretch the window by a few pixels. */
const EPIC = "crabswarm-3hp";

/** Wide enough for the two-column layouts, and narrow enough for the drawers. */
const SIZES = [
  { width: 1400, height: 800 },
  { width: 800, height: 600 },
];

let sourceId: string;
let rootId: string;

test.beforeAll(async () => {
  // Connect RPC accepts plain JSON POSTs; sources and roots are process-local
  // and keyed by path, so re-registering is idempotent daemon state.
  const source = await fetch(`${E2E_BASE_URL}/ngicks.crabswarm.issues.v1.IssuesService/AddSource`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ dir: bdFixtureDir }),
  });
  expect(source.ok).toBeTruthy();
  const sourceBody = (await source.json()) as { source: { id: string; beadsPath: string } };
  // Pins the run to the fake bd, as issues.spec.ts does: without it the daemon
  // resolves this directory to the repository's own beads database.
  expect(sourceBody.source.beadsPath).toBe("/fake/crabswarm/.beads");
  sourceId = sourceBody.source.id;

  const root = await fetch(`${E2E_BASE_URL}/ngicks.crabswarm.preview.v1.PreviewService/AddRoot`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ path: rootFixtureDir }),
  });
  expect(root.ok).toBeTruthy();
  rootId = ((await root.json()) as { root: { id: string } }).root.id;
});

function listUrl(query = ""): string {
  return `/issues/${encodeURIComponent(sourceId)}${query}`;
}

function issueUrl(issueId: string): string {
  return `/issues/${encodeURIComponent(sourceId)}/${issueId}`;
}

function docUrl(): string {
  return `/roots/${rootId}/README.md`;
}

/** Grows the page's scrolling column past any viewport, then reads what the
 *  window and the column measure. Both happen in one step so nothing renders
 *  between the two. */
function fillAndMeasure(page: Page) {
  return page.evaluate(() => {
    const column = document.querySelector("main");
    if (!column) throw new Error("the page has no scrolling column");
    const filler = document.createElement("div");
    filler.style.height = "3000px";
    column.append(filler);
    return {
      documentHeight: document.documentElement.scrollHeight,
      viewportHeight: window.innerHeight,
      columnContent: column.scrollHeight,
      columnHeight: column.clientHeight,
    };
  });
}

/** One scrollbar, and it is the column's: the window stays exactly its own
 *  height, and the column still holds and scrolls the whole document rather
 *  than having it clipped away. */
async function expectTheColumnScrolls(page: Page) {
  const m = await fillAndMeasure(page);
  // Sub-pixel layout rounding leaves the odd stray pixel.
  expect(m.documentHeight).toBeLessThanOrEqual(m.viewportHeight + 1);
  expect(m.columnContent).toBeGreaterThan(m.columnHeight);
}

for (const size of SIZES) {
  test.describe(`at ${size.width}x${size.height}`, () => {
    test.use({ viewport: size });

    test("the issue list scrolls in its column", async ({ page }) => {
      await page.goto(listUrl());
      await expect(page.getByTestId("issue-list")).toBeAttached();
      await expectTheColumnScrolls(page);
    });

    test("the labels page scrolls in its column", async ({ page }) => {
      await page.goto(`${listUrl()}/labels`);
      await expect(page.getByTestId("labels-table")).toBeAttached();
      await expectTheColumnScrolls(page);
    });

    test("the issue detail scrolls in its column, diagram and all", async ({ page }) => {
      await page.goto(issueUrl(EPIC));
      // Waiting for the drawn graph is what puts mermaid's body-level tooltip
      // on the page, so the measurement below covers it.
      await expect(page.getByTestId("graph-canvas").locator("svg")).toBeAttached();
      await expectTheColumnScrolls(page);
    });

    test("the sticky bar and the outline hold while the detail column scrolls", async ({ page }) => {
      await page.goto(issueUrl(EPIC));
      await expect(page.getByTestId("graph-canvas").locator("svg")).toBeAttached();
      await page.locator("main").evaluate((el) => el.scrollBy(0, 600));

      const bar = await page.getByTestId("detail-bar").boundingBox();
      const column = await page.locator("main").boundingBox();
      expect(bar?.y).toBeCloseTo(column?.y ?? -1, 0);
      // The outline is a second column of the detail page, drawn only at lg.
      const toc = await page.getByTestId("toc").boundingBox();
      if (toc !== null) {
        expect(toc.y).toBeGreaterThanOrEqual(0);
        expect(toc.y + toc.height).toBeLessThanOrEqual(size.height + 1);
      }
    });

    test("jumping to a section moves the column, not the window", async ({ page }) => {
      await page.goto(issueUrl(EPIC));
      await expect(page.getByTestId("graph-canvas").locator("svg")).toBeAttached();
      // What the outline and the Jump menu both call (IssueView scrollToId).
      // A window that can be scrolled at all — even by the few pixels a stray
      // node adds — gets dragged along here and stays there, taking the header
      // off the top of the screen.
      const moved = await page.evaluate(() => {
        document.getElementById("section-comments")?.scrollIntoView({ block: "start" });
        return {
          window: document.body.scrollTop + document.documentElement.scrollTop,
          column: document.querySelector("main")?.scrollTop ?? 0,
          shell: document.getElementById("app")?.getBoundingClientRect().y ?? -1,
        };
      });
      expect(moved.window).toBe(0);
      expect(moved.shell).toBe(0);
      expect(moved.column).toBeGreaterThan(0);
    });

    test("the rendered document scrolls in its column", async ({ page }) => {
      await page.goto(docUrl());
      await expect(page.locator("main pre.mermaid svg")).toBeAttached();
      await expectTheColumnScrolls(page);
    });

    test("the file tree drawer stays within the window", async ({ page }) => {
      await page.goto(docUrl());
      await expect(page.locator("main pre.mermaid svg")).toBeAttached();
      // Below lg the tree is an overlay behind the hamburger; at lg it is
      // already the persistent left column and the toggle is hidden.
      const toggle = page.locator('label[aria-label="Open file tree"]');
      if (await toggle.isVisible()) await toggle.click();
      await expect(page.getByTestId("file-tree")).toBeAttached();

      const side = await page.locator(".drawer-side").boundingBox();
      expect(side?.y).toBeGreaterThanOrEqual(0);
      expect((side?.y ?? 0) + (side?.height ?? 0)).toBeLessThanOrEqual(size.height + 1);
      expect(await page.evaluate(() => document.documentElement.scrollHeight)).toBeLessThanOrEqual(
        size.height + 1,
      );
    });
  });
}
