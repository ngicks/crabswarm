import { NavigationMenu } from "@ark-ui/react/navigation-menu";
import { useMemo, useState } from "preact/hooks";
import type {
  Issue,
  IssueComment,
  IssueEdge,
  IssueSummary,
  RenderedField,
} from "@/api/gen/ngicks/crabswarm/issues/v1/issues_service_pb.js";
import { IssueStatus } from "@/api/gen/ngicks/crabswarm/issues/v1/issues_service_pb.js";
import { commentKind, dependencyWording, metadataPairs, progressOf, useDependencies, useIssue } from "@/api/issues.js";
import { shortTime, statusBadgeClass, statusLabel } from "@/lib/format.js";
import type { GraphNode } from "@/lib/graph.js";
import { issueHref, sourceHref } from "@/lib/paths.js";
import { IssueGraph } from "./IssueGraph.js";
import { Progress } from "./IssueList.js";
import { MarkdownField, fieldAnchor } from "./MarkdownField.js";
import { Section, sectionId } from "./Section.js";

// Issue detail (GetIssue): the bead the way bd models it — summary, the four
// text fields rendered as markdown with mermaid, metadata, close reason,
// children with progress, dependencies, a local graph of the issue's
// neighbourhood and the comment thread. The view is generic over beads: the
// plan convention only shows through the `plan` label, metadata chips and the
// Decision / Discussion comment badges.

/** An issue whose summary the wire actually carried. Every sub-message of
 *  Issue is optional in proto3; the views want one settled shape. */
interface FullIssue extends Issue {
  summary: IssueSummary;
}

/** One section of the page, in page order. `field` is set for the sections
 *  that render markdown, whose own headings hang under them in the TOC. */
interface SectionEntry {
  key: string;
  title: string;
  field?: RenderedField;
}

/** A rendered field with content, or undefined: an absent message and one
 *  whose html is empty mean the same thing to the page. */
function rendered(field: RenderedField | undefined): RenderedField | undefined {
  return field && field.html !== "" ? field : undefined;
}

function fieldSections(issue: FullIssue): { key: string; title: string; field: RenderedField }[] {
  return [
    { key: "description", title: "Description", field: rendered(issue.description) },
    { key: "design", title: "Design", field: rendered(issue.design) },
    { key: "acceptance", title: "Acceptance criteria", field: rendered(issue.acceptanceCriteria) },
    { key: "notes", title: "Notes", field: rendered(issue.notes) },
  ].filter((s): s is { key: string; title: string; field: RenderedField } => s.field !== undefined);
}

function closeReasonOf(issue: FullIssue): RenderedField | undefined {
  return issue.summary.status === IssueStatus.CLOSED ? rendered(issue.closeReason) : undefined;
}

/** Every section the page actually renders, in page order: the outline the TOC
 *  and the jump menu both read, so neither can drift from the page. */
function outlineOf(issue: FullIssue): SectionEntry[] {
  const entries: SectionEntry[] = [];
  const close = closeReasonOf(issue);
  if (close) entries.push({ key: "close", title: "Close reason", field: close });
  entries.push(...fieldSections(issue));
  if (issue.children.length > 0) entries.push({ key: "children", title: `Children (${issue.children.length})` });
  if (issue.dependencies.length > 0) {
    entries.push({ key: "dependencies", title: `Dependencies (${issue.dependencies.length})` });
  }
  if (neighbourIds(issue).length > 1) entries.push({ key: "neighbourhood", title: "Neighbourhood" });
  if (issue.comments.length > 0) entries.push({ key: "comments", title: `Comments (${issue.comments.length})` });
  return entries;
}

function scrollToId(id: string): void {
  document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

export function IssueView({
  sourceId,
  issueId,
  search,
  listing,
}: {
  sourceId: string;
  issueId: string;
  search: string;
  /** The source's listing, where the neighbourhood graph finds a neighbour's
   *  title and status: IssueDependency carries neither. */
  listing: IssueSummary[];
}) {
  const { data, isLoading, error } = useIssue(sourceId, issueId);
  const issue = data?.issue;

  if (isLoading) {
    return (
      <div class="p-6">
        <span class="loading loading-spinner" />
      </div>
    );
  }
  if (error || !issue?.summary) {
    return <div class="p-6 text-sm opacity-60">No issue {issueId} in this source.</div>;
  }
  const full = issue as FullIssue;

  const sections = fieldSections(full);
  const outline = outlineOf(full);
  const close = closeReasonOf(full);

  return (
    <div class="flex min-w-0 gap-4">
      <div class="min-w-0 flex-1">
        <DetailBar issue={full} sourceId={sourceId} search={search} outline={outline} />
        <div class="space-y-4 pt-4">
          <Header issue={full} sourceId={sourceId} search={search} />
          {close && (
            <Section id="close" title="Close reason">
              <MarkdownField field={close} prefix="close" />
            </Section>
          )}
          {sections.map((s) => (
            <Section key={s.key} id={s.key} title={s.title}>
              <MarkdownField field={s.field} prefix={s.key} />
            </Section>
          ))}
          <Children issue={full} sourceId={sourceId} search={search} />
          <Dependencies issue={full} sourceId={sourceId} search={search} />
          <Neighbourhood issue={full} sourceId={sourceId} search={search} listing={listing} />
          <Comments issue={full} />
        </div>
      </div>
      <Toc outline={outline} />
    </div>
  );
}

// The bar above the detail column, sticky at the top of the scrollport: where
// the reader came from, which issue this is, and one menu to reach any section
// of a long issue without scrolling for it.
//
// The menu is Ark's navigation menu with the hover trigger disabled, so it
// opens on a click and stays open until a link is chosen (it is controlled: a
// link scrolls and then clears the value). Its content is anchored to the item
// rather than portalled — a single popup needs no shared viewport.
function DetailBar({
  issue,
  sourceId,
  search,
  outline,
}: {
  issue: FullIssue;
  sourceId: string;
  search: string;
  outline: SectionEntry[];
}) {
  const [open, setOpen] = useState("");

  return (
    <div
      // A sticky element pins to the scroll container's content box, not to
      // its padding edge, so a plain `top-0` would leave the page's own
      // padding above the bar for content to scroll through. The negative top
      // and the negative side margins pull the bar over that padding instead.
      class="sticky -top-4 z-20 -mx-4 flex items-center gap-3 border-b border-base-content/25 bg-base-100 px-4 py-2 sm:-top-6 sm:-mx-6 sm:px-6"
      data-testid="detail-bar"
    >
      <a class="link link-hover shrink-0 text-sm opacity-70" href={sourceHref(sourceId, search)}>
        ← back to the list
      </a>
      <span class="shrink-0 font-mono text-xs opacity-70">{issue.summary.id}</span>
      <span class="truncate text-sm font-semibold">{issue.summary.title}</span>

      <NavigationMenu.Root
        class="ml-auto shrink-0"
        value={open}
        onValueChange={(d) => setOpen(d.value)}
        disableHoverTrigger
        disablePointerLeaveClose
      >
        <NavigationMenu.List class="m-0 list-none p-0">
          <NavigationMenu.Item value="jump" class="relative">
            <NavigationMenu.Trigger class="btn btn-sm btn-ghost" data-testid="jump-menu-trigger">
              Jump to ▾
            </NavigationMenu.Trigger>
            <NavigationMenu.Content
              class="menu absolute right-0 top-full z-30 w-64 flex-nowrap rounded-box border border-base-content/25 bg-base-100 shadow"
              data-testid="jump-menu-content"
            >
              {outline.map((e) => (
                <li key={e.key}>
                  <NavigationMenu.Link
                    href={`#${sectionId(e.key)}`}
                    data-testid={`jump-to-${e.key}`}
                    onClick={(ev) => {
                      ev.preventDefault();
                      scrollToId(sectionId(e.key));
                      setOpen("");
                    }}
                  >
                    <span class="truncate">{e.title}</span>
                  </NavigationMenu.Link>
                </li>
              ))}
            </NavigationMenu.Content>
          </NavigationMenu.Item>
        </NavigationMenu.List>
      </NavigationMenu.Root>
    </div>
  );
}

function Header({ issue, sourceId, search }: { issue: FullIssue; sourceId: string; search: string }) {
  const s = issue.summary;
  const metadata = metadataPairs(issue.metadataJson);
  const progress = progressOf(s);

  return (
    <div class="rounded-box border border-base-content/25 bg-base-100 p-5 shadow-sm">
      <h1 class="text-3xl font-semibold leading-tight">{s.title}</h1>

      <div class="mt-3 flex flex-wrap items-center gap-2 text-xs">
        <span class={`badge ${statusBadgeClass(s.status)}`}>{statusLabel(s.status)}</span>
        <span class="font-mono opacity-70">{s.id}</span>
        <span class="badge badge-outline whitespace-nowrap">{s.issueType}</span>
        <span class="opacity-60">priority {s.priority}</span>
        {s.labels.map((l) => (
          <span key={l} class="badge badge-ghost whitespace-nowrap">
            {l}
          </span>
        ))}
      </div>

      <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs opacity-70">
        {s.parentId && (
          <span>
            parent{" "}
            <a class="link font-mono" href={issueHref(sourceId, s.parentId, search)}>
              {s.parentId}
            </a>
          </span>
        )}
        <span>created {shortTime(s.createdAt)}</span>
        <span>updated {shortTime(s.updatedAt)}</span>
        {s.childCount > 0 && <span>{s.childCount} children</span>}
        {s.commentCount > 0 && <span>{s.commentCount} comments</span>}
      </div>

      {metadata.length > 0 && (
        <div class="mt-2 flex flex-wrap gap-1" data-testid="metadata">
          {metadata.map(([k, v]) => (
            <span key={k} class="badge badge-outline whitespace-nowrap font-mono">
              {k}={v}
            </span>
          ))}
        </div>
      )}

      {progress && <Progress closed={progress.closed} total={progress.total} />}
    </div>
  );
}

function Children({ issue, sourceId, search }: { issue: FullIssue; sourceId: string; search: string }) {
  if (issue.children.length === 0) return null;
  return (
    <Section id="children" title={`Children (${issue.children.length})`}>
      <div class="overflow-x-auto">
        <table class="table text-sm">
          <thead>
            <tr>
              <th class="w-40">id</th>
              <th class="w-32">status</th>
              <th>title</th>
            </tr>
          </thead>
          <tbody>
            {issue.children.map((c) => (
              <tr key={c.id} class="hover">
                <td class="font-mono text-xs">
                  <a class="link" href={issueHref(sourceId, c.id, search)}>
                    {c.id}
                  </a>
                </td>
                <td>
                  <span class={`badge badge-sm ${statusBadgeClass(c.status)}`}>{statusLabel(c.status)}</span>
                </td>
                <td>
                  <a class="link link-hover" href={issueHref(sourceId, c.id, search)}>
                    {c.title}
                  </a>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Section>
  );
}

function Dependencies({ issue, sourceId, search }: { issue: FullIssue; sourceId: string; search: string }) {
  if (issue.dependencies.length === 0) return null;
  return (
    <Section id="dependencies" title={`Dependencies (${issue.dependencies.length})`}>
      <div class="overflow-x-auto">
        <table class="table text-sm">
          <thead>
            <tr>
              <th class="w-36">relation</th>
              <th class="w-36">type</th>
              <th class="w-40">id</th>
              <th>title</th>
            </tr>
          </thead>
          <tbody>
            {issue.dependencies.map((d) => (
              <tr key={`${d.type}-${d.id}-${String(d.outgoing)}`} class="hover">
                <td class="text-xs">{dependencyWording(d)}</td>
                <td>
                  <span class="badge badge-outline badge-sm whitespace-nowrap">{d.type}</span>
                </td>
                <td class="font-mono text-xs">
                  <a class="link" href={issueHref(sourceId, d.id, search)}>
                    {d.id}
                  </a>
                </td>
                <td class="text-sm">{d.title}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Section>
  );
}

// The issue with everything one edge away: parent, children, dependencies.
// Edges among the neighbours themselves are drawn too when the source has
// them, since ListDependencies returns every edge inside the given set.
function neighbourIds(issue: FullIssue): string[] {
  const s = issue.summary;
  const ids = new Set<string>([s.id]);
  if (s.parentId !== "") ids.add(s.parentId);
  for (const c of issue.children) ids.add(c.id);
  for (const d of issue.dependencies) ids.add(d.id);
  return [...ids];
}

/** The graph and the queries feeding it must keep their identity across a
 *  render that changed nothing, or the mermaid effect throws the drawing away
 *  and redraws it on every keystroke elsewhere on the page. */
const NO_EDGES: IssueEdge[] = [];
const NO_IDS: string[] = [];

function Neighbourhood({
  issue,
  sourceId,
  search,
  listing,
}: {
  issue: FullIssue;
  sourceId: string;
  search: string;
  listing: IssueSummary[];
}) {
  const ids = useMemo(() => neighbourIds(issue), [issue]);
  // A neighbour's title and status come from the listing; a child's from the
  // issue itself, so an epic still draws before the listing lands.
  const nodes = useMemo(() => {
    const known = new Map<string, IssueSummary>();
    for (const s of listing) known.set(s.id, s);
    for (const c of issue.children) known.set(c.id, c);
    known.set(issue.summary.id, issue.summary);
    const out: GraphNode[] = [];
    for (const id of ids) {
      const n = known.get(id);
      if (!n) continue;
      out.push({ id: n.id, title: n.title, status: n.status, current: n.id === issue.summary.id });
    }
    return out;
  }, [ids, issue, listing]);
  // An issue standing alone draws no graph, and every bd read is a subprocess,
  // so it must not ask for the edges of a set of one either.
  const { data } = useDependencies(sourceId, ids.length > 1 ? ids : NO_IDS);

  if (ids.length <= 1) return null;

  // IssueGraph draws its own section card: the legend and the zoom toolbar
  // belong in the header strip beside the title.
  return (
    <IssueGraph
      sourceId={sourceId}
      nodes={nodes}
      edges={data?.edges ?? NO_EDGES}
      search={search}
      sectionKey="neighbourhood"
      testId="local-graph"
    />
  );
}

function Comments({ issue }: { issue: FullIssue }) {
  if (issue.comments.length === 0) return null;
  return (
    <Section id="comments" title={`Comments (${issue.comments.length})`}>
      <div class="divide-y divide-base-300" data-testid="comments">
        {issue.comments.map((c, n) => (
          <CommentEntry key={c.id} comment={c} index={n} />
        ))}
      </div>
    </Section>
  );
}

function CommentEntry({ comment, index }: { comment: IssueComment; index: number }) {
  const kind = commentKind(comment);
  const text = rendered(comment.text);
  return (
    <div>
      <div class="flex flex-wrap items-center gap-2 bg-base-200 px-5 py-2 text-xs opacity-70">
        <span class="font-medium">{comment.author}</span>
        {kind !== "" && (
          <span class={`badge ${kind === "Decision" ? "badge-primary" : "badge-secondary"}`}>{kind}</span>
        )}
        <span>{shortTime(comment.createdAt)}</span>
      </div>
      {text && <MarkdownField field={text} prefix={`comment-${index}`} class="px-5 py-4" />}
    </div>
  );
}

// Right column: the page outline — every section, and under the rendered
// fields their own headings, straight off the toc arrays the renderer
// produced. Heading anchors are namespaced per field, matching MarkdownField.
//
// `top-6` clears the sticky detail bar, and `flex-nowrap` keeps the daisyUI
// menu from compressing a long outline into the aside's height instead of
// letting the aside scroll it.
function Toc({ outline }: { outline: SectionEntry[] }) {
  if (outline.length === 0) return null;

  return (
    <aside
      class="sticky top-6 hidden h-[calc(100dvh_-_8rem)] w-[240px] shrink-0 self-start overflow-auto border-l border-base-content/25 pl-2 lg:block"
      data-testid="toc"
    >
      <div class="px-2 py-2 text-xs font-semibold uppercase tracking-wide opacity-60">On this page</div>
      <ul class="menu menu-sm w-full flex-nowrap p-0">
        {outline.map((e) => (
          <li key={e.key}>
            <a
              href={`#${sectionId(e.key)}`}
              onClick={(ev) => {
                ev.preventDefault();
                scrollToId(sectionId(e.key));
              }}
            >
              <span class="truncate font-medium">{e.title}</span>
            </a>
            {e.field !== undefined && <Headings entry={e} field={e.field} />}
          </li>
        ))}
      </ul>
    </aside>
  );
}

function Headings({ entry, field }: { entry: SectionEntry; field: RenderedField }) {
  const headings = field.toc.filter((h) => h.id !== "" && h.level <= 3);
  if (headings.length === 0) return null;
  return (
    <ul class="flex-nowrap">
      {headings.map((h) => (
        <li key={`${entry.key}-${h.id}`}>
          <a
            href={`#${fieldAnchor(entry.key, h.id)}`}
            style={{ paddingLeft: `${(h.level - 1) * 10 + 8}px` }}
            onClick={(ev) => {
              ev.preventDefault();
              scrollToId(fieldAnchor(entry.key, h.id));
            }}
          >
            <span class="truncate">{h.text}</span>
          </a>
        </li>
      ))}
    </ul>
  );
}
