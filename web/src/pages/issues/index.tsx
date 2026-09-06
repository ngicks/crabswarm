// The Issues surface: /issues and everything under it. The tab and its routes
// exist so the header can be complete, but the screens themselves — the source
// switcher, the issue list, the detail page — are not built yet.
export function IssuesPage() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">
      <div class="mx-auto max-w-2xl space-y-4">
        <h1 class="text-2xl font-bold">Issues</h1>
        <div class="alert">
          <span>The Issues tab is coming. Nothing is served here yet.</span>
        </div>
      </div>
    </main>
  );
}
