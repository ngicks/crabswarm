import type { Issue, IssueComment, RenderedField } from "@/api/client.js";
import { getIssue, listDependencies } from "@/api/client.js";
import { commentKind, dependencyWording, metadataPairs, progressOf } from "@/api/issues.js";
import type { GraphNode } from "@/lib/graph.js";
import { shortTime, statusBadgeClass, statusLabel } from "@/lib/format.js";
import { issueHref } from "@/lib/paths.js";
import { IssueGraph } from "./IssueGraph.js";
import { Progress } from "./IssueList.js";
import { MarkdownField, fieldAnchor } from "./MarkdownField.js";
import { useOpenIssue } from "./useIssues.js";

// Issue detail (GetIssue): the bead the way bd models it — summary, the four
// text fields rendered as markdown with mermaid, metadata, close reason,
// children with progress, dependencies, a local graph of the issue's
// neighbourhood (D15) and the comment thread (D4: generic over beads, not a
// plan-specific view; the plan convention only shows through the `plan`
// label, the `idea_gate` metadata chip and the Decision/Discussion badges).

export function IssueView({ sourceId, issueId, search }: { sourceId: string; issueId: string; search: string }) {
  const issue = useOpenIssue(sourceId, issueId);

  if (!issue) {
    return <div class="p-6 text-sm opacity-60">No issue {issueId} in this source.</div>;
  }

  const sections: { key: string; title: string; field: RenderedField }[] = [
    { key: "description", title: "Description", field: issue.description },
    { key: "design", title: "Design", field: issue.design },
    { key: "acceptance", title: "Acceptance criteria", field: issue.acceptanceCriteria },
    { key: "notes", title: "Notes", field: issue.notes },
  ].filter((s) => s.field.html !== "");

  return (
    <div class="flex min-w-0 gap-4">
      <div class="min-w-0 flex-1 space-y-4">
        <Header issue={issue} sourceId={sourceId} search={search} />
        {sections.map((s) => (
          <section key={s.key}>
            <h2 class="mb-1 px-1 text-xs font-semibold uppercase tracking-wide opacity-60">{s.title}</h2>
            <MarkdownField field={s.field} prefix={s.key} />
          </section>
        ))}
        <Children issue={issue} sourceId={sourceId} search={search} />
        <Dependencies issue={issue} sourceId={sourceId} search={search} />
        <Neighbourhood issue={issue} sourceId={sourceId} search={search} />
        <Comments issue={issue} />
      </div>
      <Toc issue={issue} />
    </div>
  );
}

function Header({ issue, sourceId, search }: { issue: Issue; sourceId: string; search: string }) {
  const s = issue.summary;
  const metadata = metadataPairs(issue.metadataJson);
  const closed = s.status === "ISSUE_STATUS_CLOSED";
  const progress = progressOf(s);

  return (
    <div class="rounded-box border border-base-300 bg-base-100 p-5 shadow-sm">
      <div class="mb-1 flex flex-wrap items-center gap-2 text-xs">
        <span class="font-mono opacity-70">{s.id}</span>
        <span class="badge badge-outline badge-sm">{s.issueType}</span>
        <span class={`badge badge-sm ${statusBadgeClass(s.status)}`}>{statusLabel(s.status)}</span>
        <span class="opacity-60">priority {s.priority}</span>
        {s.labels.map((l) => (
          <span key={l} class="badge badge-ghost badge-sm">
            {l}
          </span>
        ))}
      </div>

      <h1 class="text-xl font-bold">{s.title}</h1>

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

      {progress && <Progress closed={progress.closed} total={progress.total} />}

      {metadata.length > 0 && (
        <div class="mt-2 flex flex-wrap gap-1" data-testid="metadata">
          {metadata.map(([k, v]) => (
            <span key={k} class="badge badge-outline badge-sm font-mono text-xs">
              {k}={v}
            </span>
          ))}
        </div>
      )}

      {closed && issue.closeReason.html !== "" && (
        <div class="mt-3">
          <h2 class="mb-1 text-xs font-semibold uppercase tracking-wide opacity-60">Close reason</h2>
          <MarkdownField field={issue.closeReason} prefix="close" />
        </div>
      )}
    </div>
  );
}

function Children({ issue, sourceId, search }: { issue: Issue; sourceId: string; search: string }) {
  if (issue.children.length === 0) return null;
  return (
    <section>
      <h2 class="mb-1 px-1 text-xs font-semibold uppercase tracking-wide opacity-60">
        Children ({issue.children.length})
      </h2>
      <div class="overflow-x-auto rounded-box border border-base-300 bg-base-100 shadow-sm">
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
                  <span class={`badge badge-xs ${statusBadgeClass(c.status)}`}>{statusLabel(c.status)}</span>
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
    </section>
  );
}

function Dependencies({ issue, sourceId, search }: { issue: Issue; sourceId: string; search: string }) {
  if (issue.dependencies.length === 0) return null;
  return (
    <section>
      <h2 class="mb-1 px-1 text-xs font-semibold uppercase tracking-wide opacity-60">
        Dependencies ({issue.dependencies.length})
      </h2>
      <div class="overflow-x-auto rounded-box border border-base-300 bg-base-100 shadow-sm">
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
                  <span class="badge badge-outline badge-xs">{d.type}</span>
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
    </section>
  );
}

// The issue with everything one edge away: parent, children, dependencies.
// Edges among the neighbours themselves are drawn too when the source has
// them, since ListDependencies returns every edge inside the given set.
function Neighbourhood({ issue, sourceId, search }: { issue: Issue; sourceId: string; search: string }) {
  const s = issue.summary;
  const ids = new Set<string>([s.id]);
  if (s.parentId !== "") ids.add(s.parentId);
  for (const c of issue.children) ids.add(c.id);
  for (const d of issue.dependencies) ids.add(d.id);
  if (ids.size === 1) return null;

  const nodes: GraphNode[] = [];
  for (const id of ids) {
    const n = id === s.id ? s : getIssue(sourceId, id)?.summary;
    if (!n) continue;
    nodes.push({ id: n.id, title: n.title, status: n.status, current: n.id === s.id });
  }
  const edges = listDependencies(sourceId, [...ids]);

  return (
    <section>
      <h2 class="mb-1 px-1 text-xs font-semibold uppercase tracking-wide opacity-60">Neighbourhood</h2>
      <IssueGraph sourceId={sourceId} nodes={nodes} edges={edges} search={search} testId="local-graph" />
    </section>
  );
}

function Comments({ issue }: { issue: Issue }) {
  if (issue.comments.length === 0) return null;
  return (
    <section>
      <h2 class="mb-1 px-1 text-xs font-semibold uppercase tracking-wide opacity-60">
        Comments ({issue.comments.length})
      </h2>
      <div class="space-y-3" data-testid="comments">
        {issue.comments.map((c, n) => (
          <CommentEntry key={c.id} comment={c} index={n} />
        ))}
      </div>
    </section>
  );
}

function CommentEntry({ comment, index }: { comment: IssueComment; index: number }) {
  const kind = commentKind(comment);
  return (
    <div>
      <div class="mb-1 flex flex-wrap items-center gap-2 px-1 text-xs opacity-70">
        {kind !== "" && (
          <span class={`badge badge-sm ${kind === "Decision" ? "badge-primary" : "badge-secondary"}`}>{kind}</span>
        )}
        <span class="font-medium">{comment.author}</span>
        <span>{shortTime(comment.createdAt)}</span>
      </div>
      <MarkdownField field={comment.text} prefix={`comment-${index}`} />
    </div>
  );
}

// Right column: the description and design headings, straight off the toc
// arrays the renderer produced. Anchors are namespaced per field, matching
// MarkdownField.
function Toc({ issue }: { issue: Issue }) {
  const entries = [
    ...issue.description.toc.map((h) => ({ ...h, prefix: "description" })),
    ...issue.design.toc.map((h) => ({ ...h, prefix: "design" })),
  ].filter((h) => h.id !== "" && h.level <= 3);
  if (entries.length === 0) return null;

  return (
    <aside class="sticky top-0 hidden h-[calc(100dvh_-_6rem)] w-[240px] shrink-0 self-start overflow-auto border-l border-base-300 pl-2 lg:block">
      <div class="px-2 py-2 text-xs font-semibold uppercase tracking-wide opacity-60">On this page</div>
      <ul class="menu menu-sm w-full p-0">
        {entries.map((h) => (
          <li key={`${h.prefix}-${h.id}`}>
            <a
              href={`#${fieldAnchor(h.prefix, h.id)}`}
              style={{ paddingLeft: `${(h.level - 1) * 10 + 8}px` }}
              onClick={(e) => {
                e.preventDefault();
                document
                  .getElementById(fieldAnchor(h.prefix, h.id))
                  ?.scrollIntoView({ behavior: "smooth", block: "start" });
              }}
            >
              <span class="truncate">{h.text}</span>
            </a>
          </li>
        ))}
      </ul>
    </aside>
  );
}
