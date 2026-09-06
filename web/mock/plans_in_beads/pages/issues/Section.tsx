import type { ComponentChildren } from "preact";

// One section of the issue detail page: a card whose header strip carries the
// section name, so the eye can skim where a section starts and ends. Every
// section on the page uses it — rendered markdown, the children and dependency
// tables, the neighbourhood graph and the comment thread.
//
// It lives in its own file rather than in IssueView: IssueGraph draws its own
// strip (the legend and the zoom toolbar belong in it) and IssueView imports
// IssueGraph, so sharing through IssueView would make the two files a cycle.

export function Section({
  title,
  extra,
  testId,
  children,
}: {
  title: string;
  extra?: ComponentChildren;
  testId?: string;
  children: ComponentChildren;
}) {
  return (
    <section class="overflow-hidden rounded-box border border-base-300 bg-base-100 shadow-sm" data-testid={testId}>
      {/* The strip is base-300, and so is the card border, so its bottom edge
          gets its own colour: in light theme base-300 alone sits too close to
          the base-100 body to mark the boundary at a glance. */}
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 border-b-2 border-base-content/15 bg-base-300 px-5 py-3">
        <h2 class="text-lg font-semibold">{title}</h2>
        {extra}
      </div>
      {children}
    </section>
  );
}
