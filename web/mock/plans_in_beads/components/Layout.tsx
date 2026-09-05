import type { ComponentChildren } from "preact";
import { useLocation } from "preact-iso";
import { sourceIdOf } from "@/lib/paths.js";
import { Header } from "./Header.js";

// The frame every page renders inside: the tab header, then the routed page
// filling the rest of the column.
//
// Unlike the app's Layout.tsx this is not daisyUI's drawer: the tab header
// spans the full page width, above both columns, and the drawer's sidebar is a
// 100dvh sticky block. The pages that have a left column own it themselves
// (pages/issues/index.tsx), on bg-base-200 with the switcher above the list,
// the way Layout.tsx / RootSwitcher.tsx / FileTree.tsx stack them.
export function Layout({ children }: { children: ComponentChildren }) {
  const loc = useLocation();
  const tab: "roots" | "issues" = loc.path.startsWith("/roots") ? "roots" : "issues";

  return (
    <div class="flex h-full min-h-full flex-col">
      <Header tab={tab} sourceId={sourceIdOf(loc.path)} />
      {children}
    </div>
  );
}
