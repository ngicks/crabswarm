// URLs of the issues surface, following PLAN.md "SPA routes":
// /issues/{sourceId}[/{issueId}], with both ids percent-encoded. The app's
// counterpart is docHref / parseDocLocation in web/src/routes.tsx.

export function issueHref(sourceId: string, issueId: string): string {
  return `/issues/${encodeURIComponent(sourceId)}/${encodeURIComponent(issueId)}`;
}

export function sourceHref(sourceId: string): string {
  return `/issues/${encodeURIComponent(sourceId)}`;
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
