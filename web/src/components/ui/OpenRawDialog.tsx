import { Dialog } from "@ark-ui/react/dialog";
import { Portal } from "@ark-ui/react/portal";
import { signal } from "@preact/signals";
import { rawUrl } from "@/api/client.js";

// A single app-level confirm dialog (Ark UI Dialog, headless; daisyUI skin).
// Clicking a non-markdown tree entry sets the target; confirming opens the raw
// bytes in a new tab and lets the browser do its default thing (DECISION D9).
interface RawTarget {
  rootId: string;
  path: string;
  name: string;
}

const target = signal<RawTarget | null>(null);

export function openRaw(rootId: string, path: string, name: string): void {
  target.value = { rootId, path, name };
}

export function OpenRawDialog() {
  const t = target.value;
  const close = () => {
    target.value = null;
  };
  const confirm = () => {
    if (t) window.open(rawUrl(t.rootId, t.path), "_blank", "noopener");
    close();
  };

  return (
    <Dialog.Root
      open={t !== null}
      onOpenChange={(details) => {
        if (!details.open) close();
      }}
    >
      <Portal>
        <Dialog.Backdrop className="fixed inset-0 z-50 bg-black/40" />
        <Dialog.Positioner className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <Dialog.Content className="w-full max-w-sm rounded-box bg-base-100 p-5 shadow-xl">
            <Dialog.Title className="text-lg font-semibold">Open file</Dialog.Title>
            <Dialog.Description className="py-3 text-sm opacity-80">
              <span class="font-mono">{t?.name}</span> isn't a markdown document. Open it raw in a new tab?
            </Dialog.Description>
            <div class="mt-2 flex justify-end gap-2">
              <Dialog.CloseTrigger className="btn btn-ghost btn-sm">Cancel</Dialog.CloseTrigger>
              <button class="btn btn-primary btn-sm" onClick={confirm}>
                Open
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Positioner>
      </Portal>
    </Dialog.Root>
  );
}
