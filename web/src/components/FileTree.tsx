import type { JSX } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import { useTree } from "../api/queries.js";
import { EntryType, type TreeEntry } from "../gen/ngicks/crabswarm/preview/v1/preview_service_pb.js";
import { docHref, isImagePath, joinPath } from "../routes.js";
import { drawerOpen, revealTarget } from "../signals/ui.js";
import { openRaw } from "./OpenRawDialog.js";

// nvim-tree / VSCode-style lazy tree (PLAN "Frontend": FileTree). Each expanded
// directory issues its own GetTree query, so the tree is fetched one level at a
// time. All files are listed (dirs first, per the server); markdown navigates
// in the SPA, other files go through the confirm-then-raw dialog (DECISION D9).
export function FileTree({ rootId, activePath }: { rootId: string; activePath: string }) {
  return (
    <div data-testid="file-tree" class="min-h-0 flex-1 overflow-auto py-1">
      <TreeLevel rootId={rootId} dir="" depth={0} activePath={activePath} />
    </div>
  );
}

function indent(depth: number): JSX.CSSProperties {
  return { paddingLeft: `${depth * 12 + 8}px` };
}

const ROW =
  "flex w-full items-center gap-1 truncate px-2 py-1 text-left text-sm cursor-pointer";
// Selection sits on the bg-base-200 sidebar, so a base-300 tint is nearly
// invisible; use the primary color pair instead. The hover tint lives on the
// idle variant only — stacking a base-300 hover on a selected row would wash
// the highlight back out.
const ROW_IDLE = ROW + " hover:bg-base-300/70";
const ROW_SELECTED = ROW + " bg-primary text-primary-content font-medium hover:bg-primary/85";

function TreeLevel(props: { rootId: string; dir: string; depth: number; activePath: string }) {
  const { rootId, dir, depth, activePath } = props;
  const { data, isLoading, error } = useTree(rootId, dir);

  if (isLoading) {
    return (
      <div class="px-2 py-1 text-xs opacity-50" style={indent(depth)}>
        Loading…
      </div>
    );
  }
  if (error) {
    return (
      <div class="px-2 py-1 text-xs text-error" style={indent(depth)}>
        failed to load
      </div>
    );
  }

  const entries = data?.entries ?? [];
  return (
    <>
      {entries.map((e) =>
        e.type === EntryType.DIR ? (
          <DirNode key={e.name} rootId={rootId} dir={joinPath(dir, e.name)} name={e.name} depth={depth} activePath={activePath} />
        ) : (
          <FileNode key={e.name} rootId={rootId} path={joinPath(dir, e.name)} entry={e} depth={depth} activePath={activePath} />
        ),
      )}
    </>
  );
}

function DirNode(props: { rootId: string; dir: string; name: string; depth: number; activePath: string }) {
  const { rootId, dir, name, depth, activePath } = props;
  const [open, setOpen] = useState(false);
  const row = useRef<HTMLButtonElement>(null);
  // Reading the signal here subscribes this node to reveal requests.
  const target = revealTarget.value;
  const revealed = target !== null && target.rootId === rootId && target.path === dir;

  // Each level is fetched only once its parent opens, so a deep target reaches
  // its node as that node mounts — hence the same effect handles both the
  // already-mounted ancestors and the levels appearing underneath them. The
  // dependency is the target object, not its path, so re-requesting the same
  // path re-opens a directory the user collapsed in between.
  useEffect(() => {
    if (target === null || target.rootId !== rootId) return;
    if (target.path !== dir && !target.path.startsWith(dir + "/")) return;
    setOpen(true);
    if (target.path === dir) row.current?.scrollIntoView({ block: "nearest" });
  }, [target, rootId, dir]);

  return (
    <>
      <button
        ref={row}
        class={revealed ? ROW_SELECTED : ROW_IDLE}
        style={indent(depth)}
        onClick={() => setOpen((o) => !o)}
      >
        <Chevron open={open} />
        <FolderIcon />
        <span class="truncate">{name}</span>
      </button>
      {open && <TreeLevel rootId={rootId} dir={dir} depth={depth + 1} activePath={activePath} />}
    </>
  );
}

function FileNode(props: { rootId: string; path: string; entry: TreeEntry; depth: number; activePath: string }) {
  const { rootId, path, entry, depth, activePath } = props;
  const active = path === activePath;

  // Markdown and images render in-SPA (images via ImageView); other files go
  // through the confirm-then-open-raw dialog.
  if (entry.isMarkdown || isImagePath(entry.name)) {
    return (
      <a
        href={docHref(rootId, path)}
        class={active ? ROW_SELECTED : ROW_IDLE}
        style={indent(depth)}
        onClick={() => {
          drawerOpen.value = false;
        }}
      >
        <Spacer />
        <FileIcon />
        <span class="truncate">{entry.name}</span>
      </a>
    );
  }
  return (
    <button
      class={ROW_IDLE + " opacity-70"}
      style={indent(depth)}
      onClick={() => openRaw(rootId, path, entry.name)}
    >
      <Spacer />
      <FileIcon />
      <span class="truncate">{entry.name}</span>
    </button>
  );
}

function Spacer() {
  return <span class="inline-block w-[14px] shrink-0" />;
}

function Chevron({ open }: { open: boolean }) {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      class={"shrink-0 transition-transform " + (open ? "rotate-90" : "")}
    >
      <path d="M9 6l6 6-6 6" />
    </svg>
  );
}

function FolderIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0 opacity-70">
      <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
    </svg>
  );
}

function FileIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0 opacity-60">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <path d="M14 2v6h6" />
    </svg>
  );
}
