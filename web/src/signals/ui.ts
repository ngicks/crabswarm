import { signal, effect } from "@preact/signals";
// github-markdown-css ships separate light/dark stylesheets. We import them as
// inline strings and swap the active one with the theme, instead of relying on
// the package's prefers-color-scheme sheet, so the daisyUI theme and the
// document theme stay in lockstep (PLAN "Frontend": ThemeToggle syncs both).
import lightMarkdownCss from "github-markdown-css/github-markdown-light.css?inline";
import darkMarkdownCss from "github-markdown-css/github-markdown-dark.css?inline";

export type Theme = "light" | "dark";

const THEME_KEY = "crabswarm-preview-theme";

function initialTheme(): Theme {
  if (typeof localStorage !== "undefined") {
    const stored = localStorage.getItem(THEME_KEY);
    if (stored === "light" || stored === "dark") return stored;
  }
  if (typeof matchMedia !== "undefined" && matchMedia("(prefers-color-scheme: dark)").matches) {
    return "dark";
  }
  return "light";
}

/** Active color theme; drives both the daisyUI theme and the markdown body css. */
export const theme = signal<Theme>(initialTheme());

/** Whether the left file-tree drawer is open (only meaningful below `lg`). */
export const drawerOpen = signal(false);

/** Whether the right table-of-contents overlay is open. Only governs the
 * below-`lg` overlay drawer; at `lg+` the TOC column is always shown. */
export const tocOpen = signal(false);

/** A request to show (and highlight) one location in the left navigation.
 * `path` is a root-relative path; `""` means the root itself. */
export interface RevealTarget {
  rootId: string;
  path: string;
}

/** Latest reveal request, or null when nothing is being pointed at. */
export const revealTarget = signal<RevealTarget | null>(null);

/** Points the left navigation at `path` under `rootId`.
 *
 * Every call must publish a *new* object: subscribers act on its identity, so
 * that asking for the same path twice re-opens a directory the user collapsed
 * by hand in between. */
export function reveal(rootId: string, path: string): void {
  revealTarget.value = { rootId, path };
  // Below `lg` the tree lives in a closed drawer, so revealing anything there
  // is invisible unless the drawer is opened too.
  drawerOpen.value = true;
}

export function toggleTheme(): void {
  theme.value = theme.value === "dark" ? "light" : "dark";
}

const MARKDOWN_STYLE_ID = "github-markdown-theme";

// Render stylesheets served by the preview HTTP server at fixed paths
// (crabswarm/preview/httpapi/handler.go: AlertCSSPath, ChromaLightCSSPath,
// ChromaDarkCSSPath). Unlike github-markdown-css these are not bundled; they are
// linked from the server origin. alert.css is theme-independent; the chroma
// light/dark sheets swap with the theme, mirroring the markdown css swap.
const ALERT_CSS_URL = "/assets/css/alert.css";
const CHROMA_LIGHT_CSS_URL = "/assets/css/chroma-github.css";
const CHROMA_DARK_CSS_URL = "/assets/css/chroma-github-dark.css";
const ALERT_LINK_ID = "render-alert-css";
const CHROMA_LINK_ID = "render-chroma-theme";

// ensureLink installs (once) or updates a <link rel="stylesheet"> by id, so the
// chroma theme sheet can be swapped by href the same way the markdown <style>
// body is swapped by textContent.
function ensureLink(id: string, href: string): void {
  let link = document.getElementById(id) as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement("link");
    link.id = id;
    link.rel = "stylesheet";
    document.head.appendChild(link);
  }
  if (link.getAttribute("href") !== href) link.setAttribute("href", href);
}

// Reflect the theme into <html data-theme> (daisyUI) and swap the injected
// github-markdown stylesheet, and persist the choice.
effect(() => {
  const t = theme.value;
  if (typeof document === "undefined") return;

  document.documentElement.dataset.theme = t;
  document.documentElement.classList.toggle("dark", t === "dark");

  let style = document.getElementById(MARKDOWN_STYLE_ID) as HTMLStyleElement | null;
  if (!style) {
    style = document.createElement("style");
    style.id = MARKDOWN_STYLE_ID;
    document.head.appendChild(style);
  }
  style.textContent = t === "dark" ? darkMarkdownCss : lightMarkdownCss;

  // alert.css is always loaded; the chroma sheet swaps with the theme.
  ensureLink(ALERT_LINK_ID, ALERT_CSS_URL);
  ensureLink(CHROMA_LINK_ID, t === "dark" ? CHROMA_DARK_CSS_URL : CHROMA_LIGHT_CSS_URL);

  try {
    localStorage.setItem(THEME_KEY, t);
  } catch {
    // ignore storage failures (private mode, etc.)
  }
});
