// Client-side state of the issues surface, the counterpart of the app's
// signals/ui.ts. Nothing here comes off the wire; it is what the reader is
// doing right now.
import { signal } from "@preact/signals";

/** The issue the reader has open, so the simulated push can target it. */
export const openIssueId = signal<string>("");
