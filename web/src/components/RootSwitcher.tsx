import { useRoots } from "../api/queries.js";

/** Root list at the top of the left drawer (PLAN "Frontend": RootSwitcher). */
export function RootSwitcher({ activeRootId }: { activeRootId: string }) {
  const { data, isLoading, error } = useRoots();
  const roots = data?.roots ?? [];
  return (
    <div class="border-b border-base-300 p-2">
      <div class="px-2 pb-1 text-xs font-semibold uppercase tracking-wide opacity-60">Roots</div>
      {isLoading && <div class="px-2 text-xs opacity-50">Loading…</div>}
      {error && <div class="px-2 text-xs text-error">failed to load roots</div>}
      <ul class="menu menu-sm w-full p-0">
        {roots.map((r) => (
          <li key={r.id}>
            {/* Keep the drawer open on root selection (unlike file selection)
                so the user can then pick a file from the new root's tree. */}
            <a
              href={`/r/${encodeURIComponent(r.id)}/`}
              class={r.id === activeRootId ? "active" : ""}
              title={r.path}
            >
              <span class="truncate">{r.name}</span>
            </a>
          </li>
        ))}
        {roots.length === 0 && !isLoading && (
          <li class="px-2 py-1 text-xs opacity-50">
            <span>
              No roots. Run <code class="font-mono">crabswarm preview .</code>
            </span>
          </li>
        )}
      </ul>
    </div>
  );
}
