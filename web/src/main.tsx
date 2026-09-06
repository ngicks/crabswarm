import { render } from "preact";
import { LocationProvider } from "preact-iso";
import { QueryClient, QueryClientProvider } from "@tanstack/preact-query";
import "./index.css";
import "./signals/preferences.js"; // side effect: install the theme <-> <html> effect early
import { App } from "./app.js";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

const root = document.getElementById("app");
if (!root) throw new Error("missing #app mount point");

render(
  <LocationProvider>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </LocationProvider>,
  root,
);
