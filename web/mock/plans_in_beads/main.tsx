// Presentation mock of the "Issues" tab planned in
// doc/plan/2026-09-04-plans_in_beads/PLAN.md (sections "Preview integration",
// "SPA routes", "Proto"; decisions D1, D6, D7, D8, D12, D13).
//
// It is a mock, not the feature: there is no daemon, no bd, no IssuesService
// and no WatchIssues stream. What it fakes, and which plan requirements it can
// therefore NOT validate, is listed in MOCK_LIMITS.md beside this file — read
// that before drawing conclusions from what you see here.
//
// Run:  cd web && pnpm exec vite --config mock/plans_in_beads/vite.config.ts
import { render } from "preact";
import { LocationProvider } from "preact-iso";
import "./mock.css";
import "../../src/signals/ui.js"; // side effect: theme <-> <html data-theme> + markdown css
import { App } from "./App.js";

const root = document.getElementById("app");
if (!root) throw new Error("missing #app mount point");

render(
  <LocationProvider>
    <App />
  </LocationProvider>,
  root,
);
