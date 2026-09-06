// URLs of the issues surface, following PLAN.md "SPA routes":
// /issues/{sourceId}[/{issueId}][?query], with both ids percent-encoded. The
// query string (view, filters) is carried from the list onto the detail URL
// and back, so returning from an issue lands on the same view and filters.
// The app's counterpart is docHref / parseDocLocation in web/src/routes.tsx.

export function issueHref(sourceId: string, issueId: string, query = ""): string {
  return `/issues/${encodeURIComponent(sourceId)}/${encodeURIComponent(issueId)}${query}`;
}

export function sourceHref(sourceId: string, query = ""): string {
  return `/issues/${encodeURIComponent(sourceId)}${query}`;
}

/** The labels page of one source. `labels` is not a bd id, so the route can
 *  sit beside /issues/{sourceId}/{issueId} without ever shadowing an issue. */
export function labelsHref(sourceId: string, query = ""): string {
  return `/issues/${encodeURIComponent(sourceId)}/labels${query}`;
}

/** decodeURIComponent that keeps a malformed segment rather than throwing. */
export function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

/** The active source read straight off the pathname, for the header: it is
 *  rendered above the router and so cannot wait for a route to resolve. */
export function sourceIdOf(pathname: string): string {
  const m = /^\/issues\/([^/]+)/.exec(pathname);
  if (!m) return "";
  return safeDecode(m[1]);
}

/** The query string of a preact-iso location url ("" or "?..."). */
export function queryOf(url: string): string {
  const i = url.indexOf("?");
  return i < 0 ? "" : url.slice(i);
}
