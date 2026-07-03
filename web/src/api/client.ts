import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { PreviewService } from "../gen/crabpreview/v1/preview_service_pb.js";

// Same-origin transport: the SPA is served by the preview HTTP server, which
// also mounts the connect handlers. In `vite dev` the origin is the vite server
// and requests are proxied to the daemon (see vite.config.ts).
const baseUrl = typeof location !== "undefined" ? location.origin : "/";

const transport = createConnectTransport({ baseUrl });

/** Typed connect-web client for the crabpreview.v1.PreviewService. */
export const previewClient = createClient(PreviewService, transport);

/** Absolute URL to a raw file under a root, served by `GET /raw/{rootId}/{path}`. */
export function rawUrl(rootId: string, path: string): string {
  const segments = path.split("/").filter(Boolean).map(encodeURIComponent);
  return `/raw/${encodeURIComponent(rootId)}/${segments.join("/")}`;
}
