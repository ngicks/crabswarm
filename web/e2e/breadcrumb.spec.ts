import path from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test, type Locator, type Page } from "@playwright/test";
import { E2E_BASE_URL } from "../playwright.config.js";

// Drives the sticky top bar's location trail (Layout Breadcrumbs) against the
// Go preview server: only the home icon navigates, while root and directory
// crumbs point the left navigation at themselves (signals/ui reveal → FileTree
// DirNode), expanding the lazily fetched levels on the way down.
//
// The headless environment has no fonts (see mermaid-lightbox.spec.ts), so
// text-only controls — crumbs and tree rows alike — measure 0x0 and Playwright
// reports them hidden and refuses to click them. Hence `press` and attachment
// assertions instead of `click`/`toBeVisible` for those; elements with a size
// of their own (the home icon button, the drawer) are exercised normally.

const fixtureDir = path.join(path.dirname(fileURLToPath(import.meta.url)), "fixtures");
const docPath = "nested/deep/note.md";

let rootId: string;
let rootName: string;

test.beforeAll(async () => {
  const resp = await fetch(`${E2E_BASE_URL}/ngicks.crabswarm.preview.v1.PreviewService/AddRoot`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ path: fixtureDir }),
  });
  expect(resp.ok).toBeTruthy();
  // The display name is deduplicated across the daemon's roots, so read it back
  // instead of assuming the fixture dir's base name.
  const body = (await resp.json()) as { root: { id: string; name: string } };
  rootId = body.root.id;
  rootName = body.root.name;
});

function crumb(page: Page, name: string) {
  return page.getByTestId("breadcrumbs").getByRole("button").filter({ hasText: name });
}

function tree(page: Page) {
  return page.getByTestId("file-tree");
}

function press(locator: Locator) {
  return locator.dispatchEvent("click");
}

// Selection is painted with the primary pair (a base-300 tint on the base-200
// sidebar was too faint to see); the hover style mentions bg-primary/85, so
// match the whole class.
const SELECTED = /(^|\s)bg-primary(\s|$)/;

test("the trail names the root and every path segment", async ({ page }) => {
  await page.goto(`/r/${rootId}/${docPath}`);
  await expect(page.getByTestId("breadcrumbs").getByRole("button")).toHaveText([
    rootName,
    "nested",
    "deep",
    "note.md",
  ]);
});

// The root crumb reveals the root itself, which the switcher already marks as
// the active entry — so that marking has to actually be there.
test("the switcher marks the root the trail names", async ({ page }) => {
  await page.goto(`/r/${rootId}/${docPath}`);
  await press(crumb(page, rootName));
  await expect(page.getByRole("link", { name: rootName, exact: true })).toHaveClass(SELECTED);
});

test("the home icon leaves the document for the root picker", async ({ page }) => {
  await page.goto(`/r/${rootId}/${docPath}`);
  await page.getByTestId("breadcrumbs").getByRole("link", { name: "Home" }).click();
  await expect(page).toHaveURL(`${E2E_BASE_URL}/`);
  await expect(page.getByRole("heading", { name: "crabswarm preview" })).toBeAttached();
  // No root is selected any more, so there is no tree to browse.
  await expect(tree(page)).toHaveCount(0);
});

test("a directory crumb expands the tree down to it and marks it, without navigating", async ({ page }) => {
  await page.goto(`/r/${rootId}/${docPath}`);
  const dir = tree(page).getByRole("button", { name: "deep" });
  await expect(dir).toHaveCount(0);

  await press(crumb(page, "deep"));

  // Its parent had to open for the row to exist at all, and its own level too.
  await expect(dir).toBeAttached();
  await expect(dir).toHaveClass(SELECTED);
  await expect(tree(page).getByRole("link", { name: "note.md" })).toBeAttached();
  // Only the crumb's own directory is marked, not the ones passed through.
  await expect(tree(page).getByRole("button", { name: "nested" })).not.toHaveClass(SELECTED);
  await expect(page).toHaveURL(`${E2E_BASE_URL}/r/${rootId}/${docPath}`);
});

test("clicking the same crumb again re-opens a directory collapsed by hand", async ({ page }) => {
  await page.goto(`/r/${rootId}/${docPath}`);
  const child = tree(page).getByRole("button", { name: "deep" });

  await press(crumb(page, "nested"));
  await expect(child).toBeAttached();

  await press(tree(page).getByRole("button", { name: "nested" }));
  await expect(child).toHaveCount(0);

  await press(crumb(page, "nested"));
  await expect(child).toBeAttached();
});

test("below lg a crumb opens the drawer it reveals into", async ({ page }) => {
  await page.setViewportSize({ width: 500, height: 800 });
  await page.goto(`/r/${rootId}/${docPath}`);
  await expect(tree(page)).not.toBeVisible();

  await press(crumb(page, "nested"));

  await expect(tree(page)).toBeVisible();
  await expect(tree(page).getByRole("button", { name: "deep" })).toBeAttached();
});
