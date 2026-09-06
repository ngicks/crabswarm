// URLs of the file browser: /roots/{rootId}/{path...}. The file path may
// contain slashes, so instead of relying on the router's wildcard shape we
// parse it out of the pathname directly (and encode it symmetrically for
// links). rootId and every path segment are percent-encoded in the URL.

export interface DocLocation {
  rootId: string;
  path: string;
}

/** decodeURIComponent that keeps a malformed segment rather than throwing. */
export function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

export function parseDocLocation(pathname: string): DocLocation {
  const m = pathname.match(/^\/roots\/([^/]+)(?:\/(.*))?$/);
  if (!m) return { rootId: "", path: "" };
  const rest = m[2] ?? "";
  const path = rest.split("/").filter(Boolean).map(safeDecode).join("/");
  return { rootId: safeDecode(m[1]), path };
}

export function docHref(rootId: string, path: string): string {
  const segs = path.split("/").filter(Boolean).map(encodeURIComponent);
  return `/roots/${encodeURIComponent(rootId)}/${segs.join("/")}`;
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
