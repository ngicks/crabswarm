import { useEffect, useState } from "preact/hooks";
import { useDocument } from "../api/queries.js";
import { tocOpen } from "../signals/ui.js";

// Right-panel table of contents with active-heading highlight (PLAN "Frontend":
// Toc). It reuses the GetDocument query (same key as DocView, so no extra
// fetch) for the heading list, and an IntersectionObserver over the rendered
// heading anchors — whose ids match the TOC ids via the server AutoHeadingID
// pass — to track which heading is in view.
export function Toc({ rootId, path }: { rootId: string; path: string }) {
  const { data } = useDocument(rootId, path);
  const toc = data?.toc ?? [];
  const [activeId, setActiveId] = useState("");

  useEffect(() => {
    if (toc.length === 0) return;
    const visible = new Set<string>();
    const obs = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) visible.add(entry.target.id);
          else visible.delete(entry.target.id);
        }
        const first = toc.find((h) => visible.has(h.id));
        if (first) setActiveId(first.id);
      },
      { rootMargin: "0px 0px -70% 0px", threshold: [0, 1] },
    );
    for (const h of toc) {
      const el = document.getElementById(h.id);
      if (el) obs.observe(el);
    }
    return () => obs.disconnect();
    // Re-observe after the document HTML (re)renders.
  }, [data?.html, rootId, path]);

  if (toc.length === 0) {
    return <div class="p-4 text-sm opacity-50">No headings.</div>;
  }

  return (
    <nav class="p-3 text-sm">
      <div class="mb-2 flex items-center justify-between px-2">
        <span class="text-xs font-semibold uppercase tracking-wide opacity-60">On this page</span>
        <button
          class="btn btn-ghost btn-xs lg:hidden"
          onClick={() => {
            tocOpen.value = false;
          }}
          aria-label="Close contents"
        >
          ✕
        </button>
      </div>
      <ul class="menu menu-sm w-full p-0">
        {toc.map((h) => (
          <li key={h.id}>
            <a
              href={`#${encodeURIComponent(h.id)}`}
              class={h.id === activeId ? "active" : ""}
              style={{ paddingLeft: `${(h.level - 1) * 10 + 8}px` }}
            >
              <span class="truncate">{h.text}</span>
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
