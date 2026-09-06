import { signal } from "@preact/signals";

// Where the reader is being pointed inside a page's own navigation columns.
// Unlike signals/preferences.ts nothing here is persisted: it is the state of
// the current visit, not a choice the reader made.

/** Whether the left file-tree drawer is open (only meaningful below `lg`). */
export const drawerOpen = signal(false);

/** Whether the right table-of-contents overlay is open. Only governs the
 * below-`lg` overlay drawer; at `lg+` the TOC column is always shown. */
export const tocOpen = signal(false);

/** A request to show (and highlight) one location in the left navigation.
 * `path` is a root-relative path; `""` means the root itself. */
export interface RevealTarget {
  rootId: string;
  path: string;
}

/** Latest reveal request, or null when nothing is being pointed at. */
export const revealTarget = signal<RevealTarget | null>(null);

/** Points the left navigation at `path` under `rootId`.
 *
 * Every call must publish a *new* object: subscribers act on its identity, so
 * that asking for the same path twice re-opens a directory the user collapsed
 * by hand in between. */
export function reveal(rootId: string, path: string): void {
  revealTarget.value = { rootId, path };
  // Below `lg` the tree lives in a closed drawer, so revealing anything there
  // is invisible unless the drawer is opened too.
  drawerOpen.value = true;
}
