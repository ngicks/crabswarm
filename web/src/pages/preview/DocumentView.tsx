import { useLocation } from "preact-iso";
import { useEffect, useRef } from "preact/hooks";
import { useDocument } from "@/api/preview.js";
import { rawUrl } from "@/api/client.js";
import { openLightbox, openSvgLightbox } from "@/components/Lightbox.js";
import { parseDocLocation } from "@/lib/paths.js";
import { theme } from "@/signals/preferences.js";

// DocumentView renders the pre-rendered HTML fragment from GetDocument (chroma
// classes, goldmark-mermaid `<pre class="mermaid">`, MathJax \(...\)/\[...\]
// markup) and drives the client-side enrichment the server HTML expects:
// mermaid + MathJax typesetting, relative-image rewriting to /raw, and anchor
// scrolling. The renderer emits its own CDN <script> for mermaid in client
// mode; we strip it and run our locally bundled mermaid + MathJax instead.
export function DocumentView() {
  const loc = useLocation();
  const { rootId, path } = parseDocLocation(loc.path);
  const { data, isLoading, error } = useDocument(rootId, path);
  const ref = useRef<HTMLElement>(null);
  // Read during render so the signal subscribes this component; the effect
  // below re-runs on toggle, restoring the raw `<pre class="mermaid">` source
  // via innerHTML so mermaid can re-render in the matching theme.
  const dark = theme.value === "dark";

  useEffect(() => {
    const el = ref.current;
    if (!el || !data) return;

    el.innerHTML = data.html;
    // goldmark-mermaid client mode appends a CDN <script>; drop it (scripts set
    // via innerHTML don't execute, but this keeps the DOM clean).
    el.querySelectorAll("script").forEach((s) => s.remove());
    rewriteImageSources(el, rootId, path);

    let cancelled = false;
    void enrich(el, dark).then(() => {
      if (!cancelled) scrollToHash();
    });
    return () => {
      cancelled = true;
    };
    // Re-run whenever the rendered HTML changes (live reload), the target
    // moves, or the theme flips (mermaid diagrams are theme-baked SVG).
  }, [data?.html, rootId, path, dark]);

  useEffect(() => {
    if (data?.title) document.title = `${data.title} — crabswarm preview`;
  }, [data?.title]);

  if (rootId === "") return <Placeholder text="No root selected." />;
  if (path === "") return <Placeholder text="Select a file from the tree." />;
  if (isLoading) {
    return (
      <div class="p-6">
        <span class="loading loading-spinner" />
      </div>
    );
  }
  if (error) {
    return <Placeholder text={`Failed to load: ${(error as Error).message ?? String(error)}`} />;
  }

  // Centering/width live on the wrapper, not the .markdown-body article:
  // github-markdown-css sets `.markdown-body { margin: 0 }` (injected unlayered
  // at runtime), which would otherwise beat Tailwind's `mx-auto` on the article.
  return (
    <div class="mx-auto max-w-3xl rounded-box border border-base-300 bg-base-100 p-6 shadow-sm sm:p-8">
      <article ref={ref} class="markdown-body" onClick={onArticleClick} />
    </div>
  );
}

function Placeholder({ text }: { text: string }) {
  return <div class="p-6 text-sm opacity-60">{text}</div>;
}

// --- enrichment -------------------------------------------------------------

async function enrich(el: HTMLElement, dark: boolean): Promise<void> {
  await Promise.all([runMermaid(el, dark), typesetMath(el)]);
}

let mermaidMod: Promise<typeof import("mermaid")> | null = null;

async function runMermaid(el: HTMLElement, dark: boolean): Promise<void> {
  const nodes = el.querySelectorAll<HTMLElement>("pre.mermaid, .mermaid");
  if (nodes.length === 0) return;
  mermaidMod ??= import("mermaid");
  try {
    const mermaid = (await mermaidMod).default;
    mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "default", securityLevel: "strict" });
    await mermaid.run({ nodes: Array.from(nodes) });
  } catch {
    // leave the raw diagram source visible on failure
  }
}

// MathJax is loaded once from our own origin (copied out of the `mathjax` npm
// package into /vendor/mathjax by scripts/copy-mathjax.mjs) — no CDN.
let mathjaxReady: Promise<void> | null = null;

function ensureMathJax(): Promise<void> {
  if (typeof window === "undefined") return Promise.resolve();
  const w = window as unknown as { MathJax?: MathJaxApi };
  if (w.MathJax?.typesetPromise) return Promise.resolve();
  if (mathjaxReady) return mathjaxReady;

  mathjaxReady = new Promise<void>((resolve) => {
    w.MathJax = {
      tex: { inlineMath: [["\\(", "\\)"]], displayMath: [["\\[", "\\]"]] },
      startup: {
        typeset: false,
        ready: () => {
          w.MathJax?.startup?.defaultReady?.();
          resolve();
        },
      },
    };
    const s = document.createElement("script");
    s.src = "/vendor/mathjax/tex-chtml.js";
    s.async = true;
    s.onerror = () => resolve(); // math simply won't typeset; no CDN fallback
    document.head.appendChild(s);
  });
  return mathjaxReady;
}

async function typesetMath(el: HTMLElement): Promise<void> {
  if (el.querySelector(".math") === null) return;
  await ensureMathJax();
  const w = window as unknown as { MathJax?: MathJaxApi };
  try {
    await w.MathJax?.typesetPromise?.([el]);
  } catch {
    // ignore typeset failures
  }
}

interface MathJaxApi {
  typesetPromise?: (elements?: unknown[]) => Promise<void>;
  startup?: {
    defaultReady?: () => void;
    typeset?: boolean;
    ready?: () => void;
  };
  tex?: unknown;
}

// --- images and anchors -----------------------------------------------------

function rewriteImageSources(el: HTMLElement, rootId: string, docPath: string): void {
  el.querySelectorAll<HTMLImageElement>("img").forEach((img) => {
    const src = img.getAttribute("src") ?? "";
    // Absolute URLs (http:, data:, …) and protocol-relative (//host/…) are
    // left untouched.
    if (src === "" || /^[a-z]+:/i.test(src) || src.startsWith("//") || src.startsWith("data:")) {
      return;
    }
    if (src.startsWith("/")) {
      // Root-absolute path (Zenn/GitHub `/images/foo.png` — `/` means the
      // registered root's top dir). Already-rewritten /raw/… URLs stay as-is.
      if (src.startsWith("/raw/")) return;
      img.src = rawUrl(rootId, src.slice(1));
      return;
    }
    // Relative path, resolved against the current document's directory.
    img.src = rawUrl(rootId, resolveRelative(docPath, src));
  });
}

function resolveRelative(docPath: string, rel: string): string {
  const stack = docPath.split("/").slice(0, -1);
  for (const part of rel.split("/")) {
    if (part === "" || part === ".") continue;
    if (part === "..") stack.pop();
    else stack.push(part);
  }
  return stack.join("/");
}

function onArticleClick(e: MouseEvent): void {
  // Rendered mermaid diagrams open the pan/zoom lightbox at natural size —
  // large diagrams are squeezed to the column width and become unreadable.
  // A failed render leaves raw text (no <svg>) and falls through.
  const diagram = (e.target as HTMLElement).closest<HTMLElement>("pre.mermaid, .mermaid");
  const svg = diagram?.querySelector<SVGSVGElement>("svg");
  if (svg) {
    e.preventDefault();
    openSvgLightbox(svg);
    return;
  }
  // Plain <img> (not wrapped in a link) opens the zoom lightbox; linked images
  // fall through so the anchor wins.
  const img = (e.target as HTMLElement).closest<HTMLImageElement>("img");
  if (img && !img.closest("a")) {
    e.preventDefault();
    openLightbox(img.src);
    return;
  }
  const a = (e.target as HTMLElement).closest("a");
  if (!a) return;
  const href = a.getAttribute("href") ?? "";
  if (!href.startsWith("#")) return;
  e.preventDefault();
  const id = decodeURIComponent(href.slice(1));
  const target = document.getElementById(id);
  if (target) {
    target.scrollIntoView({ behavior: "smooth", block: "start" });
    history.replaceState(null, "", href);
  }
}

function scrollToHash(): void {
  const hash = window.location.hash;
  if (hash.length <= 1) return;
  const target = document.getElementById(decodeURIComponent(hash.slice(1)));
  target?.scrollIntoView({ block: "start" });
}
