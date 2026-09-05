import { Route, Router, useLocation } from "preact-iso";
import { drawerOpen } from "../../src/signals/ui.js";
import { IssueList } from "./components/IssueList.js";
import { IssueView } from "./components/IssueView.js";
import { SourceSwitcher } from "./components/SourceSwitcher.js";
import { TabHeader } from "./components/TabHeader.js";
import { sourceById, sourceHref, sources } from "./data.js";

// Shell for the mocked SPA. Routes follow PLAN.md "SPA routes":
//
//   /                              tab header + picker for the active tab
//   /roots/{rootId}/{path...}      file browser (unchanged; a placeholder here)
//   /issues/{sourceId}             issue list for one source
//   /issues/{sourceId}/{issueId}   issue detail
//
// The shell is the file browser's: a left column on bg-base-200 with the
// switcher on top of the list (Layout.tsx / RootSwitcher.tsx / FileTree.tsx),
// content on the right. It uses a plain flex split rather than daisyUI's
// drawer because the tab header spans the full page width here, above both
// columns, and the drawer's sidebar is a 100dvh sticky block.

export function App() {
  const loc = useLocation();
  const tab: "roots" | "issues" = loc.path.startsWith("/roots") ? "roots" : "issues";

  return (
    <div class="flex h-full min-h-full flex-col">
      <TabHeader tab={tab} sourceId={sourceIdOf(loc.path)} />
      <Router>
        <Route path="/" component={Picker} />
        <Route path="/roots" component={RootsTab} />
        <Route path="/roots/:rootId/*" component={RootsTab} />
        <Route path="/issues/:sourceId" component={IssuesTab} />
        <Route path="/issues/:sourceId/:issueId" component={IssuesTab} />
        <Route default component={NotFound} />
      </Router>
    </div>
  );
}

// The header needs the active source before the router resolves a route, so it
// reads it off the pathname the way routes.tsx parses the document location.
function sourceIdOf(pathname: string): string {
  const m = /^\/issues\/([^/]+)/.exec(pathname);
  if (!m) return "";
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return m[1];
  }
}

function IssuesTab({ sourceId = "", issueId = "" }: { sourceId?: string; issueId?: string }) {
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

function RootsTab() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl space-y-4">
        <h1 class="text-2xl font-bold">Roots</h1>
        <div class="alert">
          <span>
            The file browser is unchanged by this plan. It keeps its root switcher, lazy file tree and document
            view; only its URLs move, from <code class="font-mono">/r/{"{rootId}"}/…</code> to{" "}
            <code class="font-mono">/roots/{"{rootId}"}/…</code> (D6, a breaking change the user accepted).
          </span>
        </div>
        <p class="text-sm opacity-70">
          This mock renders nothing here on purpose: it exists to show the second surface, not to re-mock the first
          one. The point of the tab header is that the two surfaces are visibly separate and share only the shell.
        </p>
        <a class="btn btn-primary btn-sm" href={sourceHref(sources[0]?.id ?? "")}>
          Back to Issues
        </a>
      </div>
    </main>
  );
}

function Picker() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl">
        <h1 class="mb-1 text-2xl font-bold">crabswarm preview — issues</h1>
        <p class="mb-4 text-sm opacity-70">
          One source per repository, keyed by its <code class="font-mono">.beads</code> path (D13).
        </p>
        <ul class="menu w-full">
          {sources.map((s) => (
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

function NotFound() {
  return (
    <main class="min-h-0 flex-1 overflow-auto bg-base-200 p-6">
      <div class="mx-auto max-w-2xl">
        <h1 class="text-xl font-bold">Not found</h1>
        <a class="link" href="/">
          Back to the source picker
        </a>
      </div>
    </main>
  );
}

function Placeholder({ text }: { text: string }) {
  return <div class="p-6 text-sm opacity-60">{text}</div>;
}

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}
