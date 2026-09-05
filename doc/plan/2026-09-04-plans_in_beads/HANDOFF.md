# Handoff

Work found while planning that this plan does not cover.

## Out-of-scope discovery — Playwright and the nix browsers disagree on the revision

`web/package.json` pins `@playwright/test` 1.61.0 and `web/playwright.config.ts`
documents that it expects chromium-1228, but `PLAYWRIGHT_BROWSERS_PATH` now
holds `chromium-1234` and `chromium_headless_shell-1234`. Playwright refuses
to launch (`Executable doesn't exist at .../chromium_headless_shell-1228/...`),
so `pnpm e2e` cannot run in this environment, and step 6's Playwright
verification depends on it. Found 2026-09-06 while driving the mock headless.

Follow-up: align the `@playwright/test` pin with the revision the nix side
provides (the user's call, per the web base preference rule: the nix side
leads, never `playwright install`). Not part of this plan.
