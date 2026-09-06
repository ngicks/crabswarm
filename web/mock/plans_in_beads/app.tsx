import { Route, Router } from "preact-iso";
import { Lightbox } from "#src/components/Lightbox.js";
import { Layout } from "./components/Layout.js";
import { NotFound } from "./pages/not-found.js";
import { LabelsPage } from "./pages/issues/LabelsPage.js";
import { IssuesPage, SourcePicker } from "./pages/issues/index.js";
import { RootsPage } from "./pages/roots/index.js";

// Shell and routing for the mocked SPA. Routes follow PLAN.md "SPA routes":
//
//   /                              tab header + picker for the active tab
//   /roots/{rootId}/{path...}      file browser (unchanged; a placeholder here)
//   /issues/{sourceId}             issue list for one source
//   /issues/{sourceId}/labels      the source's labels
//   /issues/{sourceId}/{issueId}   issue detail
//
// The frame around every route — the tab header over a full-width column — is
// Layout; each route below is a page under pages/.

export function App() {
  return (
    <Layout>
      <Router>
        <Route path="/" component={SourcePicker} />
        <Route path="/roots" component={RootsPage} />
        <Route path="/roots/:rootId/*" component={RootsPage} />
        <Route path="/issues/:sourceId" component={IssuesPage} />
        {/* Before the issue route: a bd id is never `labels`, so the literal
            segment can claim it. */}
        <Route path="/issues/:sourceId/labels" component={LabelsPage} />
        <Route path="/issues/:sourceId/:issueId" component={IssuesPage} />
        <Route default component={NotFound} />
      </Router>
      {/* One instance for the whole app, as routes.tsx mounts it: the overlay
          is driven by a module signal, so any rendered diagram or image can
          open it. */}
      <Lightbox />
    </Layout>
  );
}
