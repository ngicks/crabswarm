import { Router, Route, useLocation } from "preact-iso";
import { useEffect } from "preact/hooks";
import { useQueryClient } from "@tanstack/preact-query";
import { startWatchEvents } from "./api/events.js";
import { Layout } from "./components/Layout.js";
import { Lightbox } from "./components/Lightbox.js";
import { OpenRawDialog } from "./components/ui/OpenRawDialog.js";
import { IssuesPage } from "./pages/issues/index.js";
import { NotFound } from "./pages/not-found.js";
import { PreviewPage } from "./pages/preview/index.js";

// Shell and routing:
//
//   /                              root picker, Roots tab active
//   /roots                         root picker
//   /roots/{rootId}/{path...}      file browser
//   /issues, /issues/…             the issues surface
//
// Both file-browser patterns are optional-and-rest so /roots, /roots/{rootId}
// and a deep file path all reach the same page; the page reads the location
// itself rather than the matched params, because a file path carries slashes.
export function App() {
  const queryClient = useQueryClient();
  // Single live-reload stream for the whole app; invalidations refetch the
  // affected queries in place (see api/events.ts).
  useEffect(() => startWatchEvents(queryClient), [queryClient]);
  // Touch useLocation so the shell re-renders on navigation.
  useLocation();

  return (
    <Layout>
      <Router>
        <Route path="/" component={PreviewPage} />
        <Route path="/roots/:rootId?/:path*" component={PreviewPage} />
        <Route path="/issues/:rest*" component={IssuesPage} />
        <Route default component={NotFound} />
      </Router>
      <OpenRawDialog />
      <Lightbox />
    </Layout>
  );
}
