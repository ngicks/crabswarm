import { defineConfig } from "vite";
import preact from "@preact/preset-vite";
import tailwindcss from "@tailwindcss/vite";

// The preview HTTP server (pkg/crabswarm/preview) listens on 0.0.0.0:6419 by
// default (see preview.Config). In `vite dev` the SPA is served from :5173 and
// the connect-web + /raw + /healthz calls are proxied to the running daemon so
// frontend iteration needs no Go rebuild (DECISION.md D7/D7a dev path).
const env =
  (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {};
const backend = env.CRABSWARM_PREVIEW_BACKEND ?? "http://localhost:6419";

// `@` → src/, kept in sync with tsconfig.json `paths`. `#` → web/ needs nothing
// here: Vite reads package.json `imports` itself. `import.meta.url` rather than
// node:path so the config typechecks without @types/node, the same reason the
// backend URL above is read through globalThis.
const src = decodeURIComponent(new URL("./src", import.meta.url).pathname);

export default defineConfig({
  plugins: [preact(), tailwindcss()],
  resolve: { alias: { "@": src } },
  server: {
    proxy: {
      "/ngicks.crabswarm.preview.v1.PreviewService": { target: backend, changeOrigin: true },
      "/raw": { target: backend, changeOrigin: true },
      "/assets": { target: backend, changeOrigin: true },
      "/healthz": { target: backend, changeOrigin: true },
    },
  },
  build: {
    // Emitted into web/dist, which is git-ignored: `pnpm build` then packs it
    // into the committed web/dist.tar.zst that embed.go embeds.
    outDir: "dist",
    emptyOutDir: true,
  },
});
