import type { QueryClient } from "@tanstack/preact-query";
import { issuesClient, previewClient } from "./client.js";
import { qk as issuesQk } from "./issues.js";
import { qk } from "./preview.js";

const INITIAL_BACKOFF_MS = 500;
const MAX_BACKOFF_MS = 10_000;

/**
 * Consumes the PreviewService's WatchEvents server stream, mapping each event
 * to a targeted query invalidation.
 *
 * Returns a stop function; call it on teardown.
 */
export function startWatchEvents(queryClient: QueryClient): () => void {
  const healAll = () => {
    for (const key of [qk.roots(), ["tree"], ["document"]]) {
      void queryClient.invalidateQueries({ queryKey: key });
    }
  };
  return subscribe((signal) => previewClient.watchEvents({}, { signal }), healAll, (ev) => {
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
  });
}

/**
 * Consumes the IssuesService's WatchIssues server stream, produced by the
 * daemon's per-source poll over bd. The stream carries no filter, so every
 * subscriber sees every source's events.
 *
 * IssuesChanged naming no ids means "refetch this source": everything under
 * its key subtree goes. Named ids invalidate the listing (a title, a status or
 * an update stamp changed, and the listing is sorted by the last of those),
 * every dependency query of the source (an edge can have been added or
 * dropped) and each named issue's detail.
 *
 * Returns a stop function; call it on teardown.
 */
export function startWatchIssues(queryClient: QueryClient): () => void {
  const healAll = () => {
    void queryClient.invalidateQueries({ queryKey: ["issues"] });
  };
  return subscribe((signal) => issuesClient.watchIssues({}, { signal }), healAll, (ev) => {
    switch (ev.event.case) {
      case "issuesChanged": {
        const { sourceId, issueIds } = ev.event.value;
        if (issueIds.length === 0) {
          void queryClient.invalidateQueries({ queryKey: issuesQk.source(sourceId) });
          break;
        }
        void queryClient.invalidateQueries({ queryKey: issuesQk.listing(sourceId) });
        void queryClient.invalidateQueries({ queryKey: issuesQk.dependencies(sourceId) });
        for (const id of issueIds) {
          void queryClient.invalidateQueries({ queryKey: issuesQk.issue(sourceId, id) });
        }
        break;
      }
      case "sourcesChanged":
        void queryClient.invalidateQueries({ queryKey: issuesQk.sources() });
        break;
    }
  });
}

/**
 * Runs one server stream for the life of the app, handing each message to
 * `onEvent`. connect-web streaming has no native auto-reconnect (unlike SSE),
 * so the stream is re-established with exponential backoff. On every
 * (re)connect `heal` refetches what this stream feeds: events are only hints,
 * so a missed one during a drop is healed by refetching once. Healing is
 * scoped to the stream's own queries — a stream that keeps failing must not
 * drag the other surface's queries through a refetch on every retry.
 */
function subscribe<T>(
  open: (signal: AbortSignal) => AsyncIterable<T>,
  heal: () => void,
  onEvent: (ev: T) => void,
): () => void {
  const abort = new AbortController();
  let backoff = INITIAL_BACKOFF_MS;

  const run = async () => {
    while (!abort.signal.aborted) {
      try {
        heal();
        for await (const ev of open(abort.signal)) {
          backoff = INITIAL_BACKOFF_MS; // a delivered event proves the link is healthy
          onEvent(ev);
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
    // The signal lives as long as the app while a sleep lasts one backoff, so
    // the listener goes when the timer fires — a daemon that stays down means
    // one sleep per reconnect, and they must not pile up on the signal.
    const onAbort = () => {
      clearTimeout(t);
      resolve();
    };
    const t = setTimeout(() => {
      signal.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    signal.addEventListener("abort", onAbort, { once: true });
  });
}
