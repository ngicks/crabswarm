import type { QueryClient } from "@tanstack/preact-query";
import { previewClient } from "./client.js";
import { qk } from "./preview.js";

const INITIAL_BACKOFF_MS = 500;
const MAX_BACKOFF_MS = 10_000;

/**
 * Consumes the WatchEvents server stream, mapping each event to a targeted
 * query invalidation. connect-web streaming has no native auto-reconnect
 * (unlike SSE), so the stream is re-established with exponential backoff. On
 * every (re)connect we blanket-invalidate first: events are only hints, so a
 * missed event during a drop is healed by refetching everything once
 * (PLAN "Live reload" / Risks: invalidate-on-reconnect).
 *
 * Returns a stop function; call it on teardown.
 */
export function startWatchEvents(queryClient: QueryClient): () => void {
  const abort = new AbortController();
  let backoff = INITIAL_BACKOFF_MS;

  const run = async () => {
    while (!abort.signal.aborted) {
      try {
        await queryClient.invalidateQueries();
        for await (const ev of previewClient.watchEvents({}, { signal: abort.signal })) {
          backoff = INITIAL_BACKOFF_MS; // a delivered event proves the link is healthy
          switch (ev.event.case) {
            case "docChanged":
              void queryClient.invalidateQueries({
                queryKey: qk.document(ev.event.value.rootId, ev.event.value.path),
              });
              break;
            case "treeChanged":
              void queryClient.invalidateQueries({
                queryKey: qk.tree(ev.event.value.rootId, ev.event.value.dir),
              });
              break;
            case "rootsChanged":
              void queryClient.invalidateQueries({ queryKey: qk.roots() });
              break;
          }
        }
      } catch {
        // fall through to backoff+reconnect unless aborted
      }
      if (abort.signal.aborted) return;
      await sleep(backoff, abort.signal);
      backoff = Math.min(backoff * 2, MAX_BACKOFF_MS);
    }
  };

  void run();
  return () => abort.abort();
}

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const t = setTimeout(resolve, ms);
    signal.addEventListener(
      "abort",
      () => {
        clearTimeout(t);
        resolve();
      },
      { once: true },
    );
  });
}
