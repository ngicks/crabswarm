import { drawerOpen } from "#src/signals/ui.js";
import { listSources } from "@/api/client.js";
import { sourceById } from "@/api/issues.js";
import { safeDecode, sourceHref } from "@/lib/paths.js";
import { IssueList } from "./IssueList.js";
import { IssueView } from "./IssueView.js";
import { SourceSwitcher } from "./SourceSwitcher.js";

// The issues screen (PLAN.md "SPA routes"): /issues/{sourceId} lists a source,
// /issues/{sourceId}/{issueId} opens one issue beside the list, and / picks a
// source before either exists.
//
// The layout is the file browser's: a left column on bg-base-200 with the
// source switcher on top of the issue list (Layout.tsx / RootSwitcher.tsx /
// FileTree.tsx), content on the right.

export function IssuesPage({ sourceId = "", issueId = "" }: { sourceId?: string; issueId?: string }) {
  const id = safeDecode(sourceId);
  const openId = safeDecode(issueId);
  const source = sourceById(id);

  return (
    <div class="flex min-h-0 flex-1">
      <aside class="hidden w-[320px] shrink-0 flex-col border-r border-base-300 bg-base-200 text-base-content lg:flex">
        <SourceSwitcher activeSourceId={id} />
        {/* Keyed by source: the filters are per source (its labels are), so
            switching sources must not carry a label selection across. */}
        {source ? (
          <IssueList key={id} sourceId={id} activeIssueId={openId} />
        ) : (
          <div class="p-3 text-xs opacity-50">Pick a source to list its issues.</div>
        )}
      </aside>

      {drawerOpen.value && (
        <div class="fixed inset-0 z-40 lg:hidden">
          <div
            class="absolute inset-0 bg-black/40"
            onClick={() => {
              drawerOpen.value = false;
            }}
          />
          <aside class="absolute left-0 top-0 flex h-full w-[85%] max-w-[340px] flex-col bg-base-200 shadow-xl">
            <SourceSwitcher activeSourceId={id} />
            {source && <IssueList key={id} sourceId={id} activeIssueId={openId} />}
          </aside>
        </div>
      )}

      <main class="min-w-0 flex-1 overflow-auto bg-base-200 p-4 sm:p-6">
        {!source ? (
          <Placeholder text={`No source ${id} is registered.`} />
        ) : openId === "" ? (
          <Placeholder text="Select an issue from the list." />
        ) : (
          <IssueView sourceId={id} issueId={openId} />
        )}
      </main>
    </div>
  );
}

// Landing page at /: the registered sources, before one is chosen.
export function SourcePicker() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl">
        <h1 class="mb-1 text-2xl font-bold">crabswarm preview — issues</h1>
        <p class="mb-4 text-sm opacity-70">
          One source per repository, keyed by its <code class="font-mono">.beads</code> path (D13).
        </p>
        <ul class="menu w-full">
          {listSources().map((s) => (
            <li key={s.id}>
              <a href={sourceHref(s.id)}>
                <span class="font-medium">{s.prefix}</span>
                <span class="ml-2 truncate text-xs opacity-60">{s.beadsPath}</span>
              </a>
            </li>
          ))}
        </ul>
      </div>
    </main>
  );
}

function Placeholder({ text }: { text: string }) {
  return <div class="p-6 text-sm opacity-60">{text}</div>;
}
