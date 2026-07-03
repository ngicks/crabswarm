import { Router, Route, useLocation } from "preact-iso";
import { useEffect } from "preact/hooks";
import { useQueryClient } from "@tanstack/preact-query";
import { Layout } from "./components/Layout.js";
import { DocView } from "./components/DocView.js";
import { ImageView } from "./components/ImageView.js";
import { OpenRawDialog } from "./components/OpenRawDialog.js";
import { Lightbox } from "./components/Lightbox.js";
import { useRoots } from "./api/queries.js";
import { startWatchEvents } from "./api/events.js";

// --- location helpers -------------------------------------------------------
// The document view lives at /r/:rootId/<file path...>. The file path may
// contain slashes, so instead of relying on the router's wildcard shape we
// parse it out of the pathname directly (and encode it symmetrically for
// links). rootId and every path segment are percent-encoded in the URL.

export interface DocLocation {
  rootId: string;
  path: string;
}

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

export function parseDocLocation(pathname: string): DocLocation {
  const m = pathname.match(/^\/r\/([^/]+)(?:\/(.*))?$/);
  if (!m) return { rootId: "", path: "" };
  const rest = m[2] ?? "";
  const path = rest.split("/").filter(Boolean).map(safeDecode).join("/");
  return { rootId: safeDecode(m[1]), path };
}

export function docHref(rootId: string, path: string): string {
  const segs = path.split("/").filter(Boolean).map(encodeURIComponent);
  return `/r/${encodeURIComponent(rootId)}/${segs.join("/")}`;
}

export function joinPath(dir: string, name: string): string {
  return dir === "" ? name : `${dir}/${name}`;
}

// Image files render inline in the SPA (an <img> against the /raw endpoint)
// instead of routing through the confirm-then-open-raw dialog. The decision is
// purely by extension of the routed path, so no server call is needed.
const IMAGE_EXTENSIONS = new Set([
  "png",
  "jpg",
  "jpeg",
  "gif",
  "webp",
  "svg",
  "avif",
  "bmp",
  "ico",
]);

export function isImagePath(path: string): boolean {
  const dot = path.lastIndexOf(".");
  if (dot < 0) return false;
  return IMAGE_EXTENSIONS.has(path.slice(dot + 1).toLowerCase());
}

// --- routed pages -----------------------------------------------------------

function RootPicker() {
  const { data, isLoading, error } = useRoots();
  const roots = data?.roots ?? [];
  return (
    <div class="mx-auto max-w-2xl p-6">
      <h1 class="mb-4 text-2xl font-bold">crabswarm preview</h1>
      {isLoading && <span class="loading loading-spinner" />}
      {error && <div class="alert alert-error">Failed to load roots.</div>}
      {!isLoading && roots.length === 0 && (
        <div class="alert">
          <span>
            No roots registered yet. Run <code class="font-mono">crabswarm preview .</code> to add one.
          </span>
        </div>
      )}
      <ul class="menu w-full">
        {roots.map((r) => (
          <li key={r.id}>
            <a href={`/r/${encodeURIComponent(r.id)}/`}>
              <span class="font-medium">{r.name}</span>
              <span class="ml-2 truncate text-xs opacity-60">{r.path}</span>
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

// Dispatches the /r/:rootId/* route by the routed path's extension: images
// render inline (ImageView, no GetDocument), everything else is a document.
function DocRoute() {
  const loc = useLocation();
  const { path } = parseDocLocation(loc.path);
  return isImagePath(path) ? <ImageView /> : <DocView />;
}

function NotFound() {
  return (
    <div class="mx-auto max-w-2xl p-6">
      <h1 class="text-xl font-bold">Not found</h1>
      <a class="link" href="/">
        Back to roots
      </a>
    </div>
  );
}

// --- app shell --------------------------------------------------------------

export function App() {
  const queryClient = useQueryClient();
  // Single live-reload stream for the whole app; invalidations refetch the
  // affected queries in place (see api/events.ts).
  useEffect(() => startWatchEvents(queryClient), [queryClient]);
  // Touch useLocation so the shell re-renders on navigation.
  useLocation();

  return (
    <Layout>
      <Router>
        <Route path="/" component={RootPicker} />
        <Route path="/r/:rootId/*" component={DocRoute} />
        <Route default component={NotFound} />
      </Router>
      <OpenRawDialog />
      <Lightbox />
    </Layout>
  );
}
