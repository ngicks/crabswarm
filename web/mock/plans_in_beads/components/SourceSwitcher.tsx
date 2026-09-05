import { sourceHref, sources } from "../data.js";

// Issue sources at the top of the left drawer, the issues-tab counterpart of
// RootSwitcher. A source is a beads database keyed by its .beads path, so one
// repository is one entry however many worktrees it has (D13).
export function SourceSwitcher({ activeSourceId }: { activeSourceId: string }) {
  return (
    <div class="border-b border-base-300 p-2">
      <div class="px-2 pb-1 text-xs font-semibold uppercase tracking-wide opacity-60">Issue sources</div>
      <ul class="menu menu-sm w-full p-0">
        {sources.map((s) => (
          <li key={s.id}>
            <a
              href={sourceHref(s.id)}
              class={s.id === activeSourceId ? "bg-primary font-medium text-primary-content" : ""}
              title={s.beadsPath}
            >
              <span class="truncate">{s.prefix}</span>
              <span class="ml-auto font-mono text-[10px] opacity-60">{s.id.slice(0, 6)}</span>
            </a>
          </li>
        ))}
      </ul>
      <div class="px-2 pt-1 text-[11px] opacity-50">
        Registered by <code class="font-mono">crabswarm preview --issue DIR</code>
      </div>
    </div>
  );
}
