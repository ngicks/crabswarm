// Copies MathJax's self-contained `tex-chtml.js` component (TeX input + CHTML
// output, with the MathJax Modern font embedded — the non-"-nofont" variant)
// from the pinned `mathjax` npm package into web/public/vendor/mathjax, so it
// is served from our own origin with no CDN at runtime (PLAN "Frontend": bundle
// MathJax locally). vite copies public/ into dist verbatim, so the committed
// web/dist stays self-contained. Deterministic: a straight copy of one file
// from a pinned version, keeping the CI `git diff --exit-code web/dist` stable.
import { cp, mkdir, access, rm } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const webRoot = resolve(here, "..");
const srcFile = resolve(webRoot, "node_modules/mathjax/tex-chtml.js");
const dstDir = resolve(webRoot, "public/vendor/mathjax");
const dstFile = resolve(dstDir, "tex-chtml.js");

async function exists(p) {
  try {
    await access(p);
    return true;
  } catch {
    return false;
  }
}

if (!(await exists(srcFile))) {
  console.error(`[copy-mathjax] ${srcFile} not found; run \`pnpm install\` first. Skipping.`);
  process.exit(0);
}

await rm(dstDir, { recursive: true, force: true });
await mkdir(dstDir, { recursive: true });
await cp(srcFile, dstFile);
console.log(`[copy-mathjax] copied ${srcFile} -> ${dstFile}`);
