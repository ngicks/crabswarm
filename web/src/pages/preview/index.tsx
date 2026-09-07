import { useEffect } from "preact/hooks";
import { useLocation } from "preact-iso";
import { useRoots } from "@/api/preview.js";
import { docHref, isImagePath, parseDocLocation } from "@/lib/paths.js";
import { drawerOpen, reveal, revealTarget, tocOpen } from "@/signals/navigation.js";
import { DocumentView } from "./DocumentView.js";
import { FileTree } from "./FileTree.js";
import { ImageView } from "./ImageView.js";
import { RootSwitcher } from "./RootSwitcher.js";
import { Toc } from "./Toc.js";

// The file browser at /roots/{rootId}/{path...}, and the root picker at / and
// /roots. Responsive shape:
//   >= lg : persistent left sidebar (~280px) + a right TOC column (~240px).
//   < lg  : the tree is a daisyUI drawer (hamburger); the TOC a right overlay.
// Both columns hang below the app's tab header, which is why the sticky offsets
// here start at its 3rem instead of at the top of the viewport.
export function PreviewPage() {
  const loc = useLocation();
  const { rootId, path } = parseDocLocation(loc.path);

  // A revealed directory is a "you are pointing here" marker, not a selection:
  // once the user actually navigates, the active-file highlight takes over and
  // a leftover directory highlight would compete with it.
  useEffect(() => {
    revealTarget.value = null;
  }, [loc.path]);

  return (
    // daisyUI lays the drawer out as a grid and leaves its one row implicit, so
    // the row sizes to content and grows past the frame however tall the
    // document under it is — `overflow-auto` on the column below does not shrink
    // what the row asks for. Pinning the row to the grid's own height puts the
    // scrolling back where it belongs, in that column.
    <div class="drawer min-h-0 flex-1 grid-rows-[minmax(0,1fr)] lg:drawer-open">
      <input
        id="crab-left-drawer"
        type="checkbox"
        class="drawer-toggle"
        checked={drawerOpen.value}
        onChange={(e) => {
          drawerOpen.value = (e.currentTarget as HTMLInputElement).checked;
        }}
      />

      <div class="drawer-content flex h-full min-w-0 flex-col">
        <DocBar rootId={rootId} path={path} />
        <div class="flex min-h-0 flex-1">
          <main class="min-w-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">
            {rootId === "" ? <RootPicker /> : isImagePath(path) ? <ImageView /> : <DocumentView />}
          </main>
          <TocPanel rootId={rootId} path={path} />
        </div>
      </div>

      {/* daisyUI pins the sidebar to the viewport (100dvh at top 0); at lg it is
          a sticky column below the tab header, so it starts and ends 3rem lower. */}
      <div class="drawer-side z-40 lg:top-12 lg:h-[calc(100dvh_-_3rem)]">
        <label for="crab-left-drawer" aria-label="close sidebar" class="drawer-overlay" />
        <aside class="flex h-full w-[280px] flex-col bg-base-200 text-base-content">
          <RootSwitcher activeRootId={rootId} />
          {rootId !== "" ? (
            <FileTree rootId={rootId} activePath={path} />
          ) : (
            <div class="p-3 text-xs opacity-50">Pick a root to browse its files.</div>
          )}
        </aside>
      </div>
    </div>
  );
}

// Landing page at / and /roots: the registered roots, before one is chosen.
function RootPicker() {
  const { data, isLoading, error } = useRoots();
  const roots = data?.roots ?? [];
  return (
    <div class="mx-auto max-w-2xl">
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
            <a href={docHref(r.id, "")}>
              <span class="font-medium">{r.name}</span>
              <span class="ml-2 truncate text-xs opacity-60">{r.path}</span>
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

function DocBar({ rootId, path }: { rootId: string; path: string }) {
  // The TOC panel is hidden for image views (see TocPanel), so its toggle has
  // nothing to act on there — disable it too.
  const tocDisabled = rootId === "" || isImagePath(path);
  return (
    <header class="navbar sticky top-12 z-20 min-h-12 gap-1 border-b border-base-300 bg-base-100 px-2">
      <label for="crab-left-drawer" class="btn btn-square btn-ghost btn-sm lg:hidden" aria-label="Open file tree">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 6h18M3 12h18M3 18h18" />
        </svg>
      </label>
      <Breadcrumbs rootId={rootId} path={path} />
      <button
        class="btn btn-square btn-ghost btn-sm lg:hidden"
        onClick={() => {
          tocOpen.value = !tocOpen.value;
        }}
        aria-label="Toggle table of contents"
        title="Toggle contents"
        disabled={tocDisabled}
        aria-disabled={tocDisabled}
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01" />
        </svg>
      </button>
    </header>
  );
}

// Location trail for the routed document: home / root / one crumb per path
// segment. Only the home icon navigates — directories have no page of their own,
// so their crumbs point the left navigation at themselves instead (reveal).
// `py-0` keeps the trail within the bar's 3rem, which TocPanel sticks below.
function Breadcrumbs({ rootId, path }: { rootId: string; path: string }) {
  const { data } = useRoots();
  const segments = path.split("/").filter(Boolean);

  return (
    <nav aria-label="Breadcrumb" data-testid="breadcrumbs" class="breadcrumbs min-w-0 flex-1 py-0 text-sm">
      <ul>
        <li>
          <a href="/roots" class="btn btn-square btn-ghost btn-sm" aria-label="Home" title="All roots">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 10.5 12 3l9 7.5" />
              <path d="M5 9.5V21h14V9.5" />
            </svg>
          </a>
        </li>
        {rootId !== "" && (
          <li>
            {/* The roots query is already warm from RootSwitcher; the id is a
                readable-enough stand-in until it resolves. */}
            <Crumb
              rootId={rootId}
              path=""
              label={data?.roots.find((r) => r.id === rootId)?.name ?? rootId}
              current={segments.length === 0}
            />
          </li>
        )}
        {segments.map((name, i) => {
          const segPath = segments.slice(0, i + 1).join("/");
          return (
            <li key={segPath}>
              <Crumb rootId={rootId} path={segPath} label={name} current={i === segments.length - 1} />
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

function Crumb(props: { rootId: string; path: string; label: string; current: boolean }) {
  const { rootId, path, label, current } = props;
  return (
    <button
      type="button"
      class={current ? "font-medium" : ""}
      title={`Show ${label} in the file tree`}
      onClick={() => reveal(rootId, path)}
    >
      {label}
    </button>
  );
}

function TocPanel({ rootId, path }: { rootId: string; path: string }) {
  // Image views have no headings; skip the panel (and the GetDocument query the
  // Toc would otherwise fire) for them.
  if (rootId === "" || path === "" || isImagePath(path)) return null;
  return (
    <>
      {/* lg+: persistent right column, always shown. It fills the row it sits
          in — the viewport below the two stacked bars (tab header + document
          bar) — so a long TOC scrolls on its own rather than reaching past the
          frame, and `self-start` keeps it from stretching further. */}
      <aside class="hidden shrink-0 overflow-auto border-l border-base-300 lg:sticky lg:top-24 lg:block lg:h-full lg:w-[240px] lg:self-start">
        <Toc rootId={rootId} path={path} />
      </aside>

      {/* below lg: overlay drawer from the right, gated by the toggle */}
      {tocOpen.value && (
        <div class="fixed inset-0 z-40 lg:hidden">
          <div
            class="absolute inset-0 bg-black/40"
            onClick={() => {
              tocOpen.value = false;
            }}
          />
          <aside class="absolute right-0 top-0 h-full w-[80%] max-w-[300px] overflow-auto bg-base-100 shadow-xl">
            <Toc rootId={rootId} path={path} />
          </aside>
        </div>
      )}
    </>
  );
}
