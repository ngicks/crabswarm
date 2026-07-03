import { signal } from "@preact/signals";
import { useEffect } from "preact/hooks";

// App-level image lightbox (mirrors OpenRawDialog's module-signal pattern).
// Rendered images (DocView) and the standalone ImageView call openLightbox with
// an already-resolved src; the overlay toggles closed on any click or Escape.
const zoomedSrc = signal<string | null>(null);

export function openLightbox(src: string): void {
  zoomedSrc.value = src;
}

export function Lightbox() {
  const src = zoomedSrc.value;

  useEffect(() => {
    if (src === null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") zoomedSrc.value = null;
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [src]);

  if (src === null) return null;
  return (
    <div
      class="fixed inset-0 z-[60] flex cursor-zoom-out items-center justify-center bg-black/80 p-4"
      onClick={() => {
        zoomedSrc.value = null;
      }}
    >
      <img src={src} alt="" class="max-h-[90vh] max-w-[90vw] object-contain" />
    </div>
  );
}
