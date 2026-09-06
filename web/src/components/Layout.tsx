import type { ComponentChildren } from "preact";
import { useLocation } from "preact-iso";
import { Header } from "./Header.js";

// The frame every page renders inside: the tab header spanning the full page
// width, then the routed page filling the rest of the column. A page that has
// columns of its own — the file browser's tree and table of contents — owns
// them below this frame, so the header stays above both.
export function Layout({ children }: { children: ComponentChildren }) {
  const loc = useLocation();
  // `/` lands on the file browser's root picker, so anything that is not the
  // issues surface reads as the Roots tab.
  const tab: "roots" | "issues" = loc.path.startsWith("/issues") ? "issues" : "roots";

  return (
    <div class="flex h-full min-h-full flex-col">
      <Header tab={tab} />
      {children}
    </div>
  );
}
