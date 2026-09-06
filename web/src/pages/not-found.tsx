// Fallback route: a URL that is neither /roots… nor /issues….
export function NotFound() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">
      <div class="mx-auto max-w-2xl">
        <h1 class="text-xl font-bold">Not found</h1>
        <a class="link" href="/roots">
          Back to roots
        </a>
      </div>
    </main>
  );
}
