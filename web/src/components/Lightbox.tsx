import { signal } from "@preact/signals";
import { useEffect, useRef } from "preact/hooks";

// App-level lightbox with pan/zoom (mirrors OpenRawDialog's module-signal
// pattern). Two content kinds: plain images (DocView inline images, the
// standalone ImageView) and rendered mermaid SVGs (DocView diagrams), which are
// re-injected at their natural viewBox size so oversized diagrams can be read
// at 100% instead of squeezed into the article column. Wheel/buttons zoom,
// drag pans; a plain click (no drag), Escape, or the ✕ button closes.
type LightboxContent =
  | { kind: "image"; src: string }
  | { kind: "svg"; markup: string; width: number; height: number };

const content = signal<LightboxContent | null>(null);

export function openLightbox(src: string): void {
  content.value = { kind: "image", src };
}

export function openSvgLightbox(svg: SVGSVGElement): void {
  // Mermaid emits width="100%" plus a max-width inline style; pin the clone to
  // the natural viewBox size so the fit/zoom math has real pixel dimensions.
  // The clone keeps the original's ids, but the original stays in the document
  // with identical <style>/<defs>, so styling and marker refs still resolve.
  const vb = svg.viewBox.baseVal;
  const rect = svg.getBoundingClientRect();
  const width = vb.width > 0 ? vb.width : rect.width;
  const height = vb.height > 0 ? vb.height : rect.height;
  if (width <= 0 || height <= 0) return;
  const clone = svg.cloneNode(true) as SVGSVGElement;
  clone.style.maxWidth = "none";
  clone.setAttribute("width", String(width));
  clone.setAttribute("height", String(height));
  content.value = { kind: "svg", markup: clone.outerHTML, width, height };
}

export function Lightbox() {
  const c = content.value;

  useEffect(() => {
    if (c === null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") content.value = null;
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [c]);

  if (c === null) return null;
  return <Viewer c={c} />;
}

const MAX_SCALE = 16;

function Viewer({ c }: { c: LightboxContent }) {
  const overlayRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  // Pan/zoom state lives in refs and is written straight to the transform:
  // re-rendering on every pointermove/wheel event would be wasted work.
  const view = useRef({ scale: 1, tx: 0, ty: 0, fitScale: 1 });
  const dims = useRef<{ w: number; h: number } | null>(null);
  const drag = useRef<{ x: number; y: number; startX: number; startY: number; moved: boolean } | null>(null);

  const apply = (): void => {
    const el = contentRef.current;
    if (!el) return;
    const v = view.current;
    el.style.transform = `translate(${v.tx}px, ${v.ty}px) scale(${v.scale})`;
    el.style.opacity = "1";
  };

  const fit = (): void => {
    const overlay = overlayRef.current;
    const d = dims.current;
    if (!overlay || !d) return;
    const margin = 48;
    const s = Math.min((overlay.clientWidth - margin) / d.w, (overlay.clientHeight - margin) / d.h, 1);
    view.current = {
      scale: s,
      tx: (overlay.clientWidth - d.w * s) / 2,
      ty: (overlay.clientHeight - d.h * s) / 2,
      fitScale: s,
    };
    apply();
  };

  // Zoom keeping the overlay-space point (cx, cy) fixed on the content.
  const zoomAt = (cx: number, cy: number, factor: number): void => {
    const v = view.current;
    const scale = Math.min(Math.max(v.scale * factor, v.fitScale / 4), MAX_SCALE);
    const ratio = scale / v.scale;
    view.current = {
      ...v,
      scale,
      tx: cx - (cx - v.tx) * ratio,
      ty: cy - (cy - v.ty) * ratio,
    };
    apply();
  };

  const zoomAtCenter = (factor: number): void => {
    const overlay = overlayRef.current;
    if (!overlay) return;
    zoomAt(overlay.clientWidth / 2, overlay.clientHeight / 2, factor);
  };

  useEffect(() => {
    if (c.kind === "svg") {
      dims.current = { w: c.width, h: c.height };
      fit();
    } else {
      const img = imgRef.current;
      dims.current =
        img && img.complete && img.naturalWidth > 0 ? { w: img.naturalWidth, h: img.naturalHeight } : null;
      if (dims.current) fit();
      // otherwise the <img> onLoad below fits once dimensions are known
    }
  }, [c]);

  useEffect(() => {
    // Refitting on resize drops the user's zoom, but keeps the content on
    // screen, which matters more.
    window.addEventListener("resize", fit);
    return () => window.removeEventListener("resize", fit);
  }, []);

  return (
    <div
      ref={overlayRef}
      class="fixed inset-0 z-[60] cursor-grab select-none overflow-hidden bg-black/80"
      style={{ touchAction: "none" }}
      onWheel={(e) => {
        e.preventDefault();
        zoomAt(e.clientX, e.clientY, Math.exp(-e.deltaY * 0.002));
      }}
      onPointerDown={(e) => {
        if (e.button !== 0) return;
        e.preventDefault();
        overlayRef.current?.setPointerCapture(e.pointerId);
        drag.current = { x: e.clientX, y: e.clientY, startX: e.clientX, startY: e.clientY, moved: false };
      }}
      onPointerMove={(e) => {
        const d = drag.current;
        if (!d) return;
        const dx = e.clientX - d.x;
        const dy = e.clientY - d.y;
        if (Math.abs(e.clientX - d.startX) + Math.abs(e.clientY - d.startY) > 3) d.moved = true;
        d.x = e.clientX;
        d.y = e.clientY;
        view.current.tx += dx;
        view.current.ty += dy;
        apply();
      }}
      onPointerUp={() => {
        // Close only for a click the overlay tracked from pointerdown: toolbar
        // clicks stopPropagation on pointerdown but their pointerup still
        // bubbles here, and a drag is not a click.
        const d = drag.current;
        drag.current = null;
        if (d && !d.moved) content.value = null;
      }}
      onPointerCancel={() => {
        drag.current = null;
      }}
    >
      <div ref={contentRef} class="pointer-events-none absolute left-0 top-0 origin-top-left opacity-0">
        {c.kind === "image" ? (
          <img
            ref={imgRef}
            src={c.src}
            alt=""
            class="block max-w-none"
            draggable={false}
            onLoad={(e) => {
              const img = e.currentTarget;
              dims.current = { w: img.naturalWidth, h: img.naturalHeight };
              fit();
            }}
          />
        ) : (
          // bg-base-100 restores the article surface behind the diagram:
          // theme-matched mermaid output is unreadable straight on the dark
          // backdrop in light mode.
          <div
            class="bg-base-100"
            style={{ width: c.width, height: c.height }}
            dangerouslySetInnerHTML={{ __html: c.markup }}
          />
        )}
      </div>
      <div
        class="absolute right-4 top-4 flex gap-2"
        onPointerDown={(e) => {
          e.stopPropagation();
        }}
      >
        <button type="button" class="btn btn-circle btn-sm" title="Zoom out" onClick={() => zoomAtCenter(0.8)}>
          −
        </button>
        <button type="button" class="btn btn-circle btn-sm" title="Zoom in" onClick={() => zoomAtCenter(1.25)}>
          +
        </button>
        <button type="button" class="btn btn-circle btn-sm" title="Fit to screen" onClick={fit}>
          ⤢
        </button>
        <button
          type="button"
          class="btn btn-circle btn-sm"
          title="Close"
          onClick={() => {
            content.value = null;
          }}
        >
          ✕
        </button>
      </div>
      <div class="pointer-events-none absolute inset-x-0 bottom-3 text-center text-xs text-white/60">
        scroll to zoom · drag to pan · click to close
      </div>
    </div>
  );
}
