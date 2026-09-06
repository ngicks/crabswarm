import { Router, Route, useLocation } from "preact-iso";
import { useEffect } from "preact/hooks";
import { useQueryClient } from "@tanstack/preact-query";
import { startWatchEvents, startWatchIssues } from "./api/events.js";
import { Layout } from "./components/Layout.js";
import { Lightbox } from "./components/Lightbox.js";
import { OpenRawDialog } from "./components/ui/OpenRawDialog.js";
import { LabelsPage } from "./pages/issues/LabelsPage.js";
import { IssuesPage, SourcePicker } from "./pages/issues/index.js";
import { NotFound } from "./pages/not-found.js";
import { PreviewPage } from "./pages/preview/index.js";

// Shell and routing:
//
//   /                              root picker, Roots tab active
//   /roots                         root picker
//   /roots/{rootId}/{path...}      file browser
//   /issues                        issue-source picker
//   /issues/{sourceId}             issue list for one source
//   /issues/{sourceId}/labels      the source's labels
//   /issues/{sourceId}/{issueId}   issue detail
//
// Both file-browser patterns are optional-and-rest so /roots, /roots/{rootId}
// and a deep file path all reach the same page; the page reads the location
// itself rather than the matched params, because a file path carries slashes.
// The issues routes match their ids as params instead: an issue id is one
// segment, so the router can hand them over directly.
export function App() {
  const queryClient = useQueryClient();
  // One live-reload stream per surface; invalidations refetch the affected
  // queries in place (see api/events.ts).
  useEffect(() => startWatchEvents(queryClient), [queryClient]);
  useEffect(() => startWatchIssues(queryClient), [queryClient]);
  // Touch useLocation so the shell re-renders on navigation.
  useLocation();

  return (
    <Layout>
      <Router>
        <Route path="/" component={PreviewPage} />
        <Route path="/roots/:rootId?/:path*" component={PreviewPage} />
        <Route path="/issues" component={SourcePicker} />
        <Route path="/issues/:sourceId" component={IssuesPage} />
        {/* Before the issue route: a bd id is never `labels`, so the literal
            segment can claim it. */}
        <Route path="/issues/:sourceId/labels" component={LabelsPage} />
        <Route path="/issues/:sourceId/:issueId" component={IssuesPage} />
        <Route default component={NotFound} />
      </Router>
      <OpenRawDialog />
      <Lightbox />
    </Layout>
  );
}
