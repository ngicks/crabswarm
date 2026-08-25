import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test, type Page } from "@playwright/test";
import { E2E_BASE_URL } from "../playwright.config.js";

// Drives the built SPA against the Go preview server (see playwright.config
// webServer): register the fixture dir as a root, open the doc, and exercise
// the mermaid click-to-zoom lightbox (DocView.onArticleClick + Lightbox).
//
// The headless environment has no fonts, so diagram labels render empty and
// mermaid logs `translate(undefined, NaN)` attribute errors — cosmetic only;
// geometry, hit-testing and the lightbox interactions are unaffected.

const fixtureDir = path.join(path.dirname(fileURLToPath(import.meta.url)), "fixtures");

let docUrl: string;

test.beforeAll(async () => {
  // Connect RPC accepts plain JSON POSTs; roots are process-local, so
  // re-registering the same path on every run is idempotent daemon state.
  const resp = await fetch(`${E2E_BASE_URL}/crabpreview.v1.PreviewService/AddRoot`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ path: fixtureDir }),
  });
  expect(resp.ok).toBeTruthy();
  const body = (await resp.json()) as { root: { id: string } };
  docUrl = `/r/${body.root.id}/README.md`;
});

async function openLightbox(page: Page) {
  await page.goto(docUrl);
  const diagram = page.locator("pre.mermaid svg");
  await expect(diagram).toBeVisible();
  await diagram.click();
  const lightbox = page.getByTestId("lightbox");
  await expect(lightbox.locator("svg")).toBeVisible();
  return lightbox;
}

function readTransform(page: Page) {
  return page.getByTestId("lightbox-content").evaluate((el) => {
    const m = new DOMMatrixReadOnly(getComputedStyle(el).transform);
    return { scale: m.a, x: m.e, y: m.f };
  });
}

test("rendered diagram advertises zoom and opens the lightbox fitted", async ({ page }) => {
  await page.goto(docUrl);
  const pre = page.locator("pre.mermaid");
  await expect(pre.locator("svg")).toBeVisible();
  await expect(pre).toHaveCSS("cursor", "zoom-in");

  await pre.locator("svg").click();
  await expect(page.getByTestId("lightbox").locator("svg")).toBeVisible();

  // The fixture diagram is far wider than the viewport, so fit-to-screen must
  // scale it down while keeping it fully on screen.
  const t = await readTransform(page);
  expect(t.scale).toBeGreaterThan(0);
  expect(t.scale).toBeLessThan(1);
  expect(t.x).toBeGreaterThanOrEqual(0);
  expect(t.y).toBeGreaterThanOrEqual(0);
});

test("wheel zooms toward the cursor and drag pans without closing", async ({ page }) => {
  const lightbox = await openLightbox(page);
  const fitted = await readTransform(page);

  await page.mouse.move(640, 360);
  await page.mouse.wheel(0, -240);
  await expect.poll(() => readTransform(page).then((t) => t.scale)).toBeGreaterThan(fitted.scale);

  const zoomed = await readTransform(page);
  await page.mouse.move(640, 360);
  await page.mouse.down();
  await page.mouse.move(500, 300, { steps: 5 });
  await page.mouse.up();

  const panned = await readTransform(page);
  expect(panned.x).not.toBe(zoomed.x);
  expect(panned.scale).toBe(zoomed.scale);
  await expect(lightbox).toBeVisible();
});

test("toolbar zooms and refits without closing; plain click and Escape close", async ({ page }) => {
  const lightbox = await openLightbox(page);
  const fitted = await readTransform(page);

  await page.getByTitle("Zoom in").click();
  await expect.poll(() => readTransform(page).then((t) => t.scale)).toBeGreaterThan(fitted.scale);
  await expect(lightbox).toBeVisible();

  await page.getByTitle("Fit to screen").click();
  await expect.poll(() => readTransform(page).then((t) => t.scale)).toBe(fitted.scale);
  await expect(lightbox).toBeVisible();

  await page.mouse.click(640, 650);
  await expect(lightbox).not.toBeVisible();

  await openLightbox(page);
  await page.keyboard.press("Escape");
  await expect(lightbox).not.toBeVisible();
});

test("plain inline images still open the lightbox", async ({ page }) => {
  await page.goto(docUrl);
  const img = page.locator(".markdown-body img");
  await expect(img).toBeVisible();
  await img.click();
  await expect(page.getByTestId("lightbox").locator("img")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByTestId("lightbox")).not.toBeVisible();
});
