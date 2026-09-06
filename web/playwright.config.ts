import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "@playwright/test";

// The webServer rebuilds dist and `go run`s the preview server, which embeds
// the fresh dist — the suite exercises exactly the artifact that ships.
export const E2E_BASE_URL = "http://127.0.0.1:6421";

/** Recorded `bd` output the issues specs run against, relative to e2e/. */
export const BD_DATA_DIR = "fixtures/bd-data";

const here = path.dirname(fileURLToPath(import.meta.url));
// The daemon shells out to `bd` for every issue read. There is no beads
// database to point it at here, so a fake bd answering from recorded JSON goes
// first on the daemon's PATH; only the daemon sees it.
const fakeBdDir = path.join(here, "e2e", "fixtures", "bd-fake");

// Browsers come from nix (PLAYWRIGHT_BROWSERS_PATH), not `playwright install`.
// @playwright/test is pinned exactly to the version those browsers were built
// for (chromium-1228 → 1.61.x); update the nix side first before bumping it
// (see .claude/rules/ngicks.web.base-preference.md).
export default defineConfig({
  testDir: "./e2e",
  use: {
    baseURL: E2E_BASE_URL,
    // The nix chromium runs unprivileged in restricted sandboxes.
    launchOptions: { args: ["--no-sandbox"] },
  },
  webServer: {
    command: `pnpm run build && cd .. && go run ./cmd/crabswarm preview __serve --addr 127.0.0.1:6421`,
    url: `${E2E_BASE_URL}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 180_000,
    env: {
      PATH: `${fakeBdDir}${path.delimiter}${process.env.PATH ?? ""}`,
      FAKE_BD_TESTDATA: path.join(here, "e2e", BD_DATA_DIR),
    },
  },
});
