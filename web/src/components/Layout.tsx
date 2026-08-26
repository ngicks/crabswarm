import type { ComponentChildren } from "preact";
import { useEffect } from "preact/hooks";
import { useLocation } from "preact-iso";
import { drawerOpen, reveal, tocOpen, revealTarget } from "../signals/ui.js";
import { parseDocLocation, isImagePath } from "../routes.js";
import { useRoots } from "../api/queries.js";
import { RootSwitcher } from "./RootSwitcher.js";
import { FileTree } from "./FileTree.js";
import { Toc } from "./Toc.js";
import { ThemeToggle } from "./ThemeToggle.js";

// Responsive shell (PLAN "Frontend" / responsive behavior):
//   >= lg : persistent left sidebar (~280px) + toggleable right TOC (~240px).
//   < lg  : left tree is a daisyUI drawer (hamburger); TOC is a right overlay.
export function Layout({ children }: { children: ComponentChildren }) {
  const loc = useLocation();
  const { rootId, path } = parseDocLocation(loc.path);

  // A revealed directory is a "you are pointing here" marker, not a selection:
  // once the user actually navigates, the active-file highlight takes over and
  // a leftover directory highlight would compete with it.
  useEffect(() => {
    revealTarget.value = null;
  }, [loc.path]);

  return (
    <div class="drawer min-h-full lg:drawer-open">
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
        <TopBar rootId={rootId} path={path} />
        <div class="flex min-h-0 flex-1">
          <main class="min-w-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">{children}</main>
          <TocPanel rootId={rootId} path={path} />
        </div>
      </div>

      <div class="drawer-side z-40">
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

function TopBar({ rootId, path }: { rootId: string; path: string }) {
  // The TOC panel is hidden for image views (see TocPanel), so its toggle has
  // nothing to act on there — disable it too.
  const tocDisabled = rootId === "" || isImagePath(path);
  return (
    <header class="navbar sticky top-0 z-30 min-h-12 gap-1 border-b border-base-300 bg-base-100 px-2">
      <label for="crab-left-drawer" class="btn btn-square btn-ghost btn-sm lg:hidden" aria-label="Open file tree">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 6h18M3 12h18M3 18h18" />
        </svg>
      </label>
      <Breadcrumbs rootId={rootId} path={path} />
      <ThemeToggle />
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
// `py-0` keeps the trail within the header's 3rem, which TocPanel sticks below.
function Breadcrumbs({ rootId, path }: { rootId: string; path: string }) {
  const { data } = useRoots();
  const segments = path.split("/").filter(Boolean);

  return (
    <nav aria-label="Breadcrumb" data-testid="breadcrumbs" class="breadcrumbs min-w-0 flex-1 py-0 text-sm">
      <ul>
        <li>
          <a href="/" class="btn btn-square btn-ghost btn-sm" aria-label="Home" title="All roots">
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
      {/* lg+: persistent right column, always shown. Sticky just below the
          sticky header (top = header height, min-h-12 = 3rem) and capped to the
          remaining viewport so a long TOC scrolls on its own without moving the
          body; its sticky containing block is the tall flex row. */}
      <aside class="hidden shrink-0 overflow-auto border-l border-base-300 lg:sticky lg:top-12 lg:block lg:h-[calc(100dvh_-_3rem)] lg:w-[240px] lg:self-start">
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
