import { expect, test } from "@playwright/test";
import { E2E_BASE_URL } from "../playwright.config.js";

// Drives the app's tab header (components/Header) against the Go preview
// server: the two top-level surfaces, which tab reads as active where, and the
// fate of the file browser's former /r/… URLs.
//
// The headless environment has no fonts (see mermaid-lightbox.spec.ts), so
// text-only elements measure 0x0 and Playwright reports them hidden; the tabs
// carry an icon and so have a size of their own.

const ACTIVE = /(^|\s)tab-active(\s|$)/;

function tab(page: import("@playwright/test").Page, name: string) {
  return page.getByRole("tablist").getByRole("tab", { name });
}

test("the header offers both surfaces, with Roots active on the file browser", async ({ page }) => {
  await page.goto("/roots");
  await expect(tab(page, "Roots")).toHaveClass(ACTIVE);
  await expect(tab(page, "Issues")).not.toHaveClass(ACTIVE);
});

test("the landing page is the roots surface", async ({ page }) => {
  await page.goto("/");
  await expect(tab(page, "Roots")).toHaveClass(ACTIVE);
  await expect(page.getByRole("heading", { name: "crabswarm preview" })).toBeAttached();
});

test("the Issues tab navigates to its own surface and back", async ({ page }) => {
  await page.goto("/roots");
  await tab(page, "Issues").click();

  await expect(page).toHaveURL(`${E2E_BASE_URL}/issues`);
  await expect(tab(page, "Issues")).toHaveClass(ACTIVE);
  await expect(page.getByRole("heading", { name: "Issues" })).toBeAttached();

  await tab(page, "Roots").click();
  await expect(page).toHaveURL(`${E2E_BASE_URL}/roots`);
  await expect(tab(page, "Roots")).toHaveClass(ACTIVE);
});

test("the former /r/ URLs are gone", async ({ page }) => {
  await page.goto("/r/anything/README.md");
  await expect(page.getByRole("heading", { name: "Not found" })).toBeAttached();
  // The shell stays: the reader can cross to either surface from the header.
  await expect(tab(page, "Roots")).toBeAttached();
});
