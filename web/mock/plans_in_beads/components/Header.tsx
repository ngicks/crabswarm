import { ThemeToggle } from "#src/components/ThemeToggle.js";
import { listSources } from "@/api/client.js";
import { lastSimulated, simulateChange } from "@/api/events.js";
import { shortTime } from "@/lib/format.js";
import { sourceHref } from "@/lib/paths.js";
import { openIssueId } from "@/signals/issues.js";

// Top-of-page tab header (D6): the previewer grows a second surface beside the
// file browser, and the URL scheme moves with it — /roots/{rootId}/… for files,
// /issues/{sourceId}[/{issueId}] for issues (PLAN.md "SPA routes").
//
// The "simulate change" button stands in for D8's WatchIssues stream; it is
// labelled as simulated because nothing here is pushed by a daemon.
export function Header({ tab, sourceId }: { tab: "roots" | "issues"; sourceId: string }) {
  const target = sourceId || listSources()[0]?.id || "";
  const last = lastSimulated.value;

  return (
    // Lifted tabs sit on the header's bottom border: the active tab is painted
    // in the content's base-200 and overlaps the border by 1px (-mb-px), so it
    // reads as attached to the page below rather than as a toggle in the bar.
    <header class="sticky top-0 z-30 flex min-h-12 items-end gap-2 border-b border-base-300 bg-base-100 px-2">
      <label
        for="crab-left-drawer"
        class="btn btn-square btn-ghost mb-1.5 self-center lg:hidden"
        aria-label="Open the issue list"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 6h18M3 12h18M3 18h18" />
        </svg>
      </label>

      <span class="hidden self-center px-2 pb-1 text-sm font-semibold opacity-70 sm:inline">
        crabswarm preview
      </span>

      <div
        role="tablist"
        class="tabs tabs-lift -mb-px [--tab-bg:var(--color-base-200)] [--tab-border-color:var(--color-base-300)]"
      >
        <a role="tab" href="/roots" class={`tab gap-1.5 ${tab === "roots" ? "tab-active font-medium" : ""}`}>
          <FolderIcon />
          Roots
        </a>
        <a
          role="tab"
          href={sourceHref(target)}
          class={`tab gap-1.5 ${tab === "issues" ? "tab-active font-medium" : ""}`}
        >
          <IssueIcon />
          Issues
        </a>
      </div>

      <div class="flex-1" />

      {last && (
        <span class="hidden self-center pb-1 text-xs opacity-60 md:inline" data-testid="last-simulated">
          pushed {last.id} at {shortTime(last.at)}
        </span>
      )}
      <button
        class="btn btn-outline btn-sm mb-2 gap-1 self-center"
        title="Bump one issue's title and updated_at in memory — no daemon, no WatchIssues stream (D8)"
        onClick={() => simulateChange(target, openIssueId.value)}
        disabled={tab !== "issues"}
      >
        <span class="badge badge-ghost badge-sm">simulated</span>
        simulate change
      </button>
      <span class="mb-1 self-center">
        <ThemeToggle />
      </span>
    </header>
  );
}

function FolderIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
    </svg>
  );
}

function IssueIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}
