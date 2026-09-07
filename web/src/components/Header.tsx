import { ThemeToggle } from "@/components/ThemeToggle.js";

// Top-of-page tab header: the previewer has two top-level surfaces, the file
// browser under /roots/… and the issues screens under /issues/…, and the tabs
// are how a reader crosses between them. Everything specific to one surface —
// the file browser's breadcrumb trail, its drawer toggle — belongs to that
// surface's own page, below this bar.
export function Header({ tab }: { tab: "roots" | "issues" }) {
  return (
    <header class="sticky top-0 z-30 flex min-h-12 shrink-0 items-end gap-2 border-b border-base-300 bg-base-100 px-2">
      <span class="hidden self-center px-2 pb-1 text-sm font-semibold opacity-70 sm:inline">
        crabswarm preview
      </span>

      {/* Lifted tabs paint the active tab in base-100 and overlap the header's
          bottom border by 1px (-mb-px), so it reads as attached to the page
          below rather than as a toggle in the bar. */}
      <div role="tablist" class="tabs tabs-lift -mb-px">
        <a role="tab" href="/roots" class={`tab gap-1.5 ${tab === "roots" ? "tab-active font-medium" : ""}`}>
          <FolderIcon />
          Roots
        </a>
        <a role="tab" href="/issues" class={`tab gap-1.5 ${tab === "issues" ? "tab-active font-medium" : ""}`}>
          <IssueIcon />
          Issues
        </a>
      </div>

      <div class="flex-1" />

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
