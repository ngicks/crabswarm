import { defineConfig } from "vite";
import base from "../../vite.config";

// Vite config for the presentation mock. It extends the app's config so the
// mock is built by the same toolchain (preact + tailwind plugins, same
// versions, same daisyUI themes) and only moves three things:
//
//   root      this directory, so index.html here is served at /
//   resolve   `@` alias to this directory (tsconfig paths mirror it)
//   build     input is this index.html, output is a git-ignored dist/ here
//   cacheDir  likewise a git-ignored .vite/ here, not node_modules/.vite
//
// The build output deliberately never lands in web/dist: that directory is
// packed into web/dist.tar.zst and embedded in the Go binary, and a mock has
// no business anywhere near it.
//
// Run:  cd web && pnpm exec vite --config mock/plans_in_beads/vite.config.ts

// Directory of this file. `import.meta.url` rather than node:path so the
// config typechecks without @types/node (the app's tsconfig only pulls in
// vite/client), the same reason vite.config.ts reads process.env via globalThis.
const here = decodeURIComponent(new URL(".", import.meta.url).pathname);

export default defineConfig({
  ...base,
  root: here,
  // `@` → this directory, the mock's analogue of the app's `@` → src/; kept in
  // sync with tsconfig.json `paths`. `#` → web/ needs nothing here: Vite reads
  // package.json `imports` itself.
  resolve: { alias: { "@": here.replace(/\/$/, "") } },
  cacheDir: `${here}.vite`, // git-ignored (web/.gitignore: /mock/*/.vite)
  // The base config proxies /assets, /raw, /healthz and the PreviewService to a
  // running preview daemon. There is no daemon behind the mock, and the proxy
  // would swallow its own public/assets/css/*.css (the alert and chroma
  // stylesheets gen.go writes, which signals/ui.ts links) — so drop it.
  server: {},
  build: {
    rollupOptions: { input: `${here}index.html` },
    outDir: `${here}dist`, // git-ignored (web/.gitignore: /mock/*/dist), never web/dist
    emptyOutDir: true,
  },
});
