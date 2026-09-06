import { useQuery } from "@tanstack/preact-query";
import { previewClient } from "./client.js";

// Query keys are the single source of truth shared between the fetch hooks and
// the live-reload invalidation in events.ts — an event's (rootId, path) must
// produce the same key the corresponding hook subscribes to.
export const qk = {
  roots: () => ["roots"] as const,
  tree: (rootId: string, path: string) => ["tree", rootId, path] as const,
  document: (rootId: string, path: string) => ["document", rootId, path] as const,
};

export function useRoots() {
  return useQuery({
    queryKey: qk.roots(),
    queryFn: () => previewClient.listRoots({}),
  });
}

export function useTree(rootId: string, path: string, enabled = true) {
  return useQuery({
    queryKey: qk.tree(rootId, path),
    queryFn: () => previewClient.getTree({ rootId, path }),
    enabled: enabled && rootId !== "",
  });
}

export function useDocument(rootId: string, path: string) {
  return useQuery({
    queryKey: qk.document(rootId, path),
    queryFn: () => previewClient.getDocument({ rootId, path }),
    enabled: rootId !== "" && path !== "",
  });
}
