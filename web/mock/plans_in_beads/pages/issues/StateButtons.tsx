import { filterIssues } from "@/api/issues.js";
import { hasTag, toggleTag } from "@/api/query.js";

// The row above the table, as GitHub's issues page has it: "Open N" and
// "Closed N" pick one state, "Plans N" narrows to plan issues. They are
// spellings of the search bar's query — `is:open`, `is:closed`, `is:plan` —
// and their counts are what the rest of the query would match with that
// state, so the numbers answer "how many if I click".

export function StateButtons({ sourceId, q, setQ }: { sourceId: string; q: string; setQ(next: string): void }) {
  const open = hasTag(q, "is", "open");
  const closed = hasTag(q, "is", "closed");
  const plans = hasTag(q, "is", "plan");

  // Counts under each state, with the other state token swapped out.
  const withState = (state: "open" | "closed") =>
    toggleTag(toggleTag(q, "is", state === "open" ? "closed" : "open", []), "is", state, ["is:open", "is:closed"]);
  const stateOff = (state: "open" | "closed") => (hasTag(q, "is", state) ? toggleTag(q, "is", state) : q);
  const count = (query: string) => filterIssues(sourceId, query).length;
  const openCount = count(open ? q : withState("open"));
  const closedCount = count(closed ? q : withState("closed"));
  const plansCount = count(plans ? q : toggleTag(q, "is", "plan"));

  const pick = (state: "open" | "closed") => {
    // Picking the active state again clears it (everything); picking the
    // other swaps the token.
    setQ(hasTag(q, "is", state) ? stateOff(state) : withState(state));
  };

  return (
    <div class="flex flex-wrap items-center gap-1 text-sm" data-testid="state-buttons">
      <button class={`btn btn-ghost ${open ? "font-semibold" : "opacity-70"}`} onClick={() => pick("open")} data-testid="state-open">
        <OpenIcon />
        {openCount} Open
      </button>
      <button
        class={`btn btn-ghost ${closed ? "font-semibold" : "opacity-70"}`}
        onClick={() => pick("closed")}
        data-testid="state-closed"
      >
        <ClosedIcon />
        {closedCount} Closed
      </button>
      <span class="mx-1 opacity-30">|</span>
      <button
        class={`btn ${plans ? "btn-secondary" : "btn-ghost opacity-70"}`}
        onClick={() => setQ(toggleTag(q, "is", "plan"))}
        title="is:plan"
        data-testid="state-plans"
      >
        <PlanIcon />
        {plansCount} Plans
      </button>
    </div>
  );
}

function OpenIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function ClosedIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M8 12l3 3 5-6" />
    </svg>
  );
}

function PlanIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
      <path d="M4 5h16M4 12h10M4 19h7" />
    </svg>
  );
}
