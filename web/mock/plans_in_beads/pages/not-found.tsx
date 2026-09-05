// Fallback route: a URL that is neither /roots… nor /issues….
export function NotFound() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl">
        <h1 class="text-xl font-bold">Not found</h1>
        <a class="link" href="/">
          Back to the source picker
        </a>
      </div>
    </main>
  );
}
