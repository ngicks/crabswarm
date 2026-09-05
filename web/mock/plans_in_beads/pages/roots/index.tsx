import { listSources } from "@/api/client.js";
import { sourceHref } from "@/lib/paths.js";

// Placeholder for the file browser at /roots[/{rootId}/{path...}]. The mock
// does not re-render the first surface; see MOCK_LIMITS.md "No Roots tab".
export function RootsPage() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl space-y-4">
        <h1 class="text-2xl font-bold">Roots</h1>
        <div class="alert">
          <span>
            The file browser is unchanged by this plan. It keeps its root switcher, lazy file tree and document
            view; only its URLs move, from <code class="font-mono">/r/{"{rootId}"}/…</code> to{" "}
            <code class="font-mono">/roots/{"{rootId}"}/…</code> (D6, a breaking change the user accepted).
          </span>
        </div>
        <p class="text-sm opacity-70">
          This mock renders nothing here on purpose: it exists to show the second surface, not to re-mock the first
          one. The point of the tab header is that the two surfaces are visibly separate and share only the shell.
        </p>
        <a class="btn btn-primary btn-sm" href={sourceHref(listSources()[0]?.id ?? "")}>
          Back to Issues
        </a>
      </div>
    </main>
  );
}
