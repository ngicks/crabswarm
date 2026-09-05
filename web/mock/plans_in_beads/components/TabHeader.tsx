import { ThemeToggle } from "../../../src/components/ThemeToggle.js";
import { lastSimulated, openIssueId, shortTime, simulateChange, sourceHref, sources } from "../data.js";

// Top-of-page tab header (D6): the previewer grows a second surface beside the
// file browser, and the URL scheme moves with it — /roots/{rootId}/… for files,
// /issues/{sourceId}[/{issueId}] for issues (PLAN.md "SPA routes").
//
// The "simulate change" button stands in for D8's WatchIssues stream; it is
// labelled as simulated because nothing here is pushed by a daemon.
export function TabHeader({ tab, sourceId }: { tab: "roots" | "issues"; sourceId: string }) {
  const target = sourceId || sources[0]?.id || "";
  const last = lastSimulated.value;

  return (
    <header class="navbar sticky top-0 z-30 min-h-12 gap-2 border-b border-base-300 bg-base-100 px-2">
      <label
        for="crab-left-drawer"
        class="btn btn-square btn-ghost btn-sm lg:hidden"
        aria-label="Open the issue list"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 6h18M3 12h18M3 18h18" />
        </svg>
      </label>

      <span class="hidden px-2 text-sm font-semibold opacity-70 sm:inline">crabswarm preview</span>

      <div role="tablist" class="tabs tabs-box tabs-sm">
        <a role="tab" href="/roots" class={`tab ${tab === "roots" ? "tab-active" : ""}`}>
          Roots
        </a>
        <a role="tab" href={sourceHref(target)} class={`tab ${tab === "issues" ? "tab-active" : ""}`}>
          Issues
        </a>
      </div>

      <div class="flex-1" />

      {last && (
        <span class="hidden text-xs opacity-60 md:inline" data-testid="last-simulated">
          pushed {last.id} at {shortTime(last.at)}
        </span>
      )}
      <button
        class="btn btn-outline btn-xs gap-1"
        title="Bump one issue's title and updated_at in memory — no daemon, no WatchIssues stream (D8)"
        onClick={() => simulateChange(target, openIssueId.value)}
        disabled={tab !== "issues"}
      >
        <span class="badge badge-ghost badge-xs">simulated</span>
        simulate change
      </button>
      <ThemeToggle />
    </header>
  );
}
