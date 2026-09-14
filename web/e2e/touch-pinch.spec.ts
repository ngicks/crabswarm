import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test, type Page } from "@playwright/test";
import { BD_DATA_DIR, E2E_BASE_URL } from "../playwright.config.js";

// Drives the touch half of the pan/zoom surfaces against the Go preview server
// (see playwright.config webServer): the document lightbox (components/Lightbox
// over lib/gesture) and the issue detail page's dependency graph
// (pages/issues/IssueGraph). The desktop spec mermaid-lightbox.spec.ts covers
// the same surfaces with a wheel and a mouse; what only a finger reaches is a
// two-finger pinch, a two-finger pan, and a tap that has to stay a tap.
//
// Playwright's own touchscreen only taps, so a gesture with two fingers — or
// one finger that moves — goes through CDP's Input.dispatchTouchEvent, which
// reaches the page as pointer events with distinct pointerIds and
// pointerType "touch". The context declares hasTouch and isMobile so the
// `hover: none` / `pointer: coarse` media queries match the way they do on a
// phone.
//
// The headless environment has no fonts, so diagram labels render empty and
// text-only elements measure 0x0 and count as hidden; assertions read geometry,
// computed styles and text content rather than visibility.

test.use({ hasTouch: true, isMobile: true, viewport: { width: 400, height: 800 } });

const here = path.dirname(fileURLToPath(import.meta.url));
/** Real files for the document lightbox: the mermaid document and the inline image. */
const rootFixtureDir = path.join(here, "fixtures");
/** Recorded `bd` output, served by the fake bd on the daemon's PATH. */
const bdFixtureDir = path.resolve(here, BD_DATA_DIR);

/** A closed child issue whose neighbourhood is small enough to read: itself,
 *  its parent and the three issues its edges reach. */
const STEP = "crabswarm-3hp.2";
/** Its parent, the issue the first neighbour node opens. */
const EPIC = "crabswarm-3hp";

let docUrl: string;
let sourceId: string;

test.beforeAll(async () => {
  // Connect RPC accepts plain JSON POSTs; roots and sources are process-local
  // and keyed by path, so re-registering is idempotent daemon state.
  const root = await fetch(`${E2E_BASE_URL}/ngicks.crabswarm.preview.v1.PreviewService/AddRoot`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ path: rootFixtureDir }),
  });
  expect(root.ok).toBeTruthy();
  const rootBody = (await root.json()) as { root: { id: string } };
  docUrl = `/roots/${rootBody.root.id}/README.md`;

  const source = await fetch(`${E2E_BASE_URL}/ngicks.crabswarm.issues.v1.IssuesService/AddSource`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ dir: bdFixtureDir }),
  });
  expect(source.ok).toBeTruthy();
  const sourceBody = (await source.json()) as { source: { id: string; beadsPath: string } };
  // Pins the run to the fake bd: a daemon started without it on PATH resolves
  // this directory to the repository's own beads database instead.
  expect(sourceBody.source.beadsPath).toBe("/fake/crabswarm/.beads");
  sourceId = sourceBody.source.id;
});

interface Point {
  x: number;
  y: number;
}

/**
 * Moves `from[i]` to `to[i]` as one touch gesture: two points pinch, one point
 * drags. The steps are separate touchMove dispatches, each far enough to clear
 * the gesture helper's slop.
 */
async function touchGesture(page: Page, from: Point[], to: Point[], steps = 6): Promise<void> {
  const cdp = await page.context().newCDPSession(page);
  const at = (t: number) =>
    from.map((p, i) => ({
      x: p.x + (to[i].x - p.x) * t,
      y: p.y + (to[i].y - p.y) * t,
      id: i,
    }));
  await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: at(0) });
  for (let s = 1; s <= steps; s++) {
    await cdp.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: at(s / steps) });
  }
  // An empty point list lifts every finger at once.
  await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
  await cdp.detach();
}

function readTransform(page: Page) {
  return page.getByTestId("lightbox-content").evaluate((el) => {
    const m = new DOMMatrixReadOnly(getComputedStyle(el).transform);
    return { scale: m.a, x: m.e, y: m.f };
  });
}

/** The lightbox transform once the mount effect has written one. `fit()` runs
 *  in an effect, so the content reports `transform: none` for a tick after the
 *  overlay appears — and `DOMMatrixReadOnly("none")` reads as the identity,
 *  which would satisfy the checks below for the wrong reason. */
async function settledTransform(page: Page) {
  await page.waitForFunction(() => {
    const el = document.querySelector('[data-testid="lightbox-content"]');
    return el !== null && getComputedStyle(el).transform !== "none";
  });
  return readTransform(page);
}

/** The diagram once mermaid has finished drawing it. `mermaid.run()` puts an
 *  empty `<svg width="100%"><g/></svg>` in the `<pre>` first and writes the
 *  viewBox only when the layout is done. That scaffold is already visible and
 *  tappable, and the lightbox sizes itself from what it can measure: the
 *  scaffold's 100%-wide box, which fits at 1:1 — so the fit assertions read
 *  scale 1 — or, mid-swap, nothing at all, which opens no lightbox. */
async function settledDiagram(page: Page) {
  await page.waitForFunction(() => {
    const svg = document.querySelector<SVGSVGElement>("pre.mermaid svg");
    return svg !== null && svg.viewBox.baseVal.width > 0;
  });
  return page.locator("pre.mermaid svg");
}

async function openDiagram(page: Page) {
  await page.goto(docUrl);
  const diagram = await settledDiagram(page);
  await expect(diagram).toBeVisible();
  await diagram.tap();
  const lightbox = page.getByTestId("lightbox");
  await expect(lightbox.locator("svg")).toBeVisible();
  return lightbox;
}

/** The viewport-space point a pinch is centred on, and the one the content
 *  under it must still sit at afterwards. */
const MID: Point = { x: 200, y: 400 };
/** Fingers 100px apart around MID, spread to 240px apart: the same midpoint,
 *  so what was pinned between them has nowhere to drift to. */
const SPREAD_FROM: Point[] = [
  { x: 150, y: 400 },
  { x: 250, y: 400 },
];
const SPREAD_TO: Point[] = [
  { x: 80, y: 400 },
  { x: 320, y: 400 },
];

test("a tapped diagram opens fitted, worded for a finger", async ({ page }) => {
  const lightbox = await openDiagram(page);

  // The fixture diagram is far wider than the viewport, so fit-to-screen must
  // scale it down while keeping it fully on screen.
  const t = await settledTransform(page);
  expect(t.scale).toBeGreaterThan(0);
  expect(t.scale).toBeLessThan(1);
  expect(t.x).toBeGreaterThanOrEqual(0);
  expect(t.y).toBeGreaterThanOrEqual(0);

  // Both wordings sit in the DOM and a media query drops one. Neither has a
  // size to be visible by, so which one this device shows reads off `display`.
  await expect(lightbox.getByText("pinch to zoom · drag to pan · tap to close")).toHaveCSS("display", "inline");
  await expect(lightbox.getByText("scroll to zoom · drag to pan · click to close")).toHaveCSS("display", "none");
});

test("two fingers spreading zoom in around what sits between them", async ({ page }) => {
  const lightbox = await openDiagram(page);
  const before = await settledTransform(page);

  await touchGesture(page, SPREAD_FROM, SPREAD_TO);

  const after = await readTransform(page);
  expect(after.scale).toBeGreaterThan(before.scale);
  // Lifting two fingers is not a tap, however still they were held.
  await expect(lightbox).toBeVisible();

  // The point of the diagram that was under the midpoint has to still be
  // there: content coordinates out of the old transform, back through the new.
  const contentX = (MID.x - before.x) / before.scale;
  const contentY = (MID.y - before.y) / before.scale;
  expect(Math.abs(contentX * after.scale + after.x - MID.x)).toBeLessThan(4);
  expect(Math.abs(contentY * after.scale + after.y - MID.y)).toBeLessThan(4);
});

test("two fingers moving together pan without zooming", async ({ page }) => {
  await openDiagram(page);
  const before = await settledTransform(page);

  // Both fingers travel the same vector, so the spread between them never
  // changes and only the midpoint moves.
  const shift: Point = { x: 40, y: -70 };
  await touchGesture(
    page,
    SPREAD_FROM,
    SPREAD_FROM.map((p) => ({ x: p.x + shift.x, y: p.y + shift.y })),
  );

  const after = await readTransform(page);
  // Each dispatched move arrives as one pointer event per finger, so the spread
  // dips and recovers within a step and the scale comes back to itself rather
  // than staying bit-identical.
  expect(after.scale).toBeCloseTo(before.scale, 5);
  expect(Math.abs(after.x - before.x - shift.x)).toBeLessThan(4);
  expect(Math.abs(after.y - before.y - shift.y)).toBeLessThan(4);
});

test("one finger drags the diagram without closing it", async ({ page }) => {
  const lightbox = await openDiagram(page);
  const before = await settledTransform(page);

  const shift: Point = { x: 60, y: -120 };
  await touchGesture(page, [{ x: 200, y: 500 }], [{ x: 200 + shift.x, y: 500 + shift.y }]);

  const after = await readTransform(page);
  expect(after.scale).toBe(before.scale);
  expect(Math.abs(after.x - before.x - shift.x)).toBeLessThan(2);
  expect(Math.abs(after.y - before.y - shift.y)).toBeLessThan(2);
  await expect(lightbox).toBeVisible();
});

test("a tap that goes nowhere closes the lightbox", async ({ page }) => {
  const lightbox = await openDiagram(page);
  await settledTransform(page);

  await page.touchscreen.tap(MID.x, 700);
  await expect(lightbox).not.toBeVisible();
});

test("a tapped inline image opens the lightbox and pinches to zoom", async ({ page }) => {
  await page.goto(docUrl);
  // The image sits under the diagram, which shrinks from the scaffold's height
  // to the drawn one: a tap aimed before that shift lands below the image.
  await settledDiagram(page);
  const img = page.locator(".markdown-body img");
  await expect(img).toBeVisible();
  await img.tap();
  await expect(page.getByTestId("lightbox").locator("img")).toBeVisible();

  // The fixture image is smaller than the viewport, so it opens at 1:1 rather
  // than scaled down; what the pinch has to do is still grow it.
  const before = await settledTransform(page);
  await touchGesture(page, SPREAD_FROM, SPREAD_TO);
  expect((await readTransform(page)).scale).toBeGreaterThan(before.scale);
});

test("the dependency graph pinches to zoom and a node still opens its issue", async ({ page }) => {
  await page.goto(`/issues/${encodeURIComponent(sourceId)}/${STEP}`);

  const canvas = page.getByTestId("graph-canvas");
  await expect(canvas.locator("g.node")).toHaveCount(5);
  // The canvas is the drawing itself, sized to the graph and transformed; the
  // box the fingers land in is the viewport around it.
  const box = canvas.locator("..");
  await box.scrollIntoViewIfNeeded();
  const rect = await box.boundingBox();
  expect(rect).not.toBeNull();
  if (rect === null) return;

  const level = page.getByTestId("graph-zoom-level");
  await expect(level).toHaveText(/%/);
  const before = Number.parseInt((await level.textContent()) ?? "", 10);

  const cx = rect.x + rect.width / 2;
  const cy = rect.y + rect.height / 2;
  await touchGesture(
    page,
    [
      { x: cx - 50, y: cy },
      { x: cx + 50, y: cy },
    ],
    [
      { x: cx - 110, y: cy },
      { x: cx + 110, y: cy },
    ],
  );
  await expect
    .poll(async () => Number.parseInt((await level.textContent()) ?? "", 10))
    .toBeGreaterThan(before);

  // Back to the whole drawing so every node is inside the box: the point here
  // is that a pinch leaves the tap alone, not where the zoom left the nodes.
  await page.getByTestId("graph-zoom-fit").tap();
  // Node ids carry their declaration index; n0 is the issue itself and n1 the
  // first neighbour, which is its parent.
  await canvas.locator('g.node[id*="flowchart-n1-"]').tap();
  await expect(page).toHaveURL(`${E2E_BASE_URL}/issues/${encodeURIComponent(sourceId)}/${EPIC}`);
});

test("a pinch that starts on a graph node zooms and does not open it", async ({ page }) => {
  const url = `/issues/${encodeURIComponent(sourceId)}/${STEP}`;
  await page.goto(url);

  const canvas = page.getByTestId("graph-canvas");
  await expect(canvas.locator("g.node")).toHaveCount(5);
  const box = canvas.locator("..");
  await box.scrollIntoViewIfNeeded();
  const rect = await box.boundingBox();
  expect(rect).not.toBeNull();
  if (rect === null) return;

  // The graph opens centred on the issue's own node, so that is the only node
  // in the box and a tap on it routes to the URL this test already sits on. One
  // finger on the background drags a neighbour in instead: tapping that one
  // does navigate, which is what the pinch below must not end up doing.
  const start: Point = { x: rect.x + rect.width - 20, y: rect.y + 40 };
  await touchGesture(page, [start], [{ x: start.x - 300, y: start.y }]);

  const nodeRect = await canvas.locator('g.node[id*="flowchart-n3-"]').boundingBox();
  expect(nodeRect).not.toBeNull();
  if (nodeRect === null) return;
  // Where the node and the box overlap: a node hanging over an edge is only
  // touchable on the part that is still in the box.
  const left = Math.max(rect.x, nodeRect.x);
  const right = Math.min(rect.x + rect.width, nodeRect.x + nodeRect.width);
  const top = Math.max(rect.y, nodeRect.y);
  const bottom = Math.min(rect.y + rect.height, nodeRect.y + nodeRect.height);
  // Said here rather than left to the zoom assertion: a pan that failed to bring
  // the node in leaves an empty overlap, and every number below it is nonsense.
  expect(right).toBeGreaterThan(left);
  expect(bottom).toBeGreaterThan(top);
  const cx = (left + right) / 2;
  const cy = (top + bottom) / 2;
  const gap = Math.min(50, (right - left) / 4);

  const level = page.getByTestId("graph-zoom-level");
  await expect(level).toHaveText(/%/);
  const before = Number.parseInt((await level.textContent()) ?? "", 10);

  // Both fingers land on the node and end twice as far apart, so the scale
  // doubles. Landing there is the point: a touch pointer is implicitly captured
  // by the element it goes down on and the box only takes that capture once a
  // drag starts, so a pinch that begins on a node runs through the transfer —
  // which once reached the box as a lifted finger and cut the pinch short.
  await touchGesture(
    page,
    [
      { x: cx - gap, y: cy },
      { x: cx + gap, y: cy },
    ],
    [
      { x: cx - 2 * gap, y: cy },
      { x: cx + 2 * gap, y: cy },
    ],
  );

  await expect
    .poll(async () => Number.parseInt((await level.textContent()) ?? "", 10))
    .toBeGreaterThan(before * 1.5);
  await expect(page).toHaveURL(`${E2E_BASE_URL}${url}`);
});
