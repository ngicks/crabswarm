// The mock's stand-in for the WatchIssues stream (D8). The real feature
// subscribes here (web/src/api/events.ts) and lets a daemon-side poll push
// IssuesChanged; nothing is pushed to this mock, so the header's "simulate
// change" button calls simulateChange directly.
import { signal } from "@preact/signals";
import { issues } from "./client.js";

/** The most recent simulated push, shown in the header. */
export const lastSimulated = signal<{ id: string; at: string } | null>(null);

const SIMULATED_SUFFIX = / \(simulated update (\d+)\)$/;

/** Stands in for a WatchIssues push (D8): bump one issue's title and
 *  updated_at in memory. Everything reading `issues` re-renders, and the list
 *  reorders, because it is sorted newest-updated first. */
export function simulateChange(sourceId: string, preferredId: string): string | null {
  const list = issues.value;
  const target =
    list.find((i) => i.sourceId === sourceId && i.summary.id === preferredId) ??
    list.find((i) => i.sourceId === sourceId && i.summary.status === "ISSUE_STATUS_IN_PROGRESS") ??
    list.find((i) => i.sourceId === sourceId);
  if (!target) return null;

  const id = target.summary.id;
  const m = SIMULATED_SUFFIX.exec(target.summary.title);
  const bump = m ? Number(m[1]) + 1 : 1;
  const title = `${target.summary.title.replace(SIMULATED_SUFFIX, "")} (simulated update ${bump})`;
  // A pushed change must land newest, so the reordering of the list is visible.
  // The fixture's own stamps are synthetic and can sit ahead of the wall clock.
  const newest = list.reduce((acc, i) => (i.summary.updatedAt > acc ? i.summary.updatedAt : acc), "");
  const after = Date.parse(newest);
  const at = new Date(Math.max(Date.now(), Number.isNaN(after) ? 0 : after + 60_000)).toISOString();

  issues.value = list.map((i) => {
    if (i.sourceId !== sourceId) return i;
    if (i.summary.id === id) return { ...i, summary: { ...i.summary, title, updatedAt: at } };
    if (i.children.some((c) => c.id === id)) {
      return {
        ...i,
        children: i.children.map((c) => (c.id === id ? { ...c, title, updatedAt: at } : c)),
      };
    }
    return i;
  });
  lastSimulated.value = { id, at };
  return id;
}
