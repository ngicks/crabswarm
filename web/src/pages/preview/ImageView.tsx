import { useLocation } from "preact-iso";
import { useEffect } from "preact/hooks";
import { rawUrl } from "@/api/client.js";
import { openLightbox } from "@/components/Lightbox.js";
import { parseDocLocation } from "@/lib/paths.js";

// ImageView renders an image file inline (like GitHub's file view) instead of
// routing it through the confirm-then-open-raw dialog. It needs no server call:
// the routed path is an image (isImagePath in lib/paths.ts decides by extension), so
// it points an <img> straight at the existing /raw endpoint (rawUrl).
export function ImageView() {
  const loc = useLocation();
  const { rootId, path } = parseDocLocation(loc.path);
  const name = path.split("/").filter(Boolean).pop() ?? path;

  useEffect(() => {
    if (name !== "") document.title = `${name} — crabswarm preview`;
  }, [name]);

  if (rootId === "") return <Placeholder text="No root selected." />;
  if (path === "") return <Placeholder text="Select a file from the tree." />;

  return (
    <div class="mx-auto max-w-3xl rounded-box border border-base-300 bg-base-100 p-6 shadow-sm sm:p-8">
      <img
        src={rawUrl(rootId, path)}
        alt={name}
        class="mx-auto max-w-full cursor-zoom-in"
        onClick={() => openLightbox(rawUrl(rootId, path))}
      />
      <div class="mt-3 text-center font-mono text-sm opacity-60">{name}</div>
    </div>
  );
}

function Placeholder({ text }: { text: string }) {
  return <div class="p-6 text-sm opacity-60">{text}</div>;
}
