// Dependency graph as mermaid text (D15): the graph view and the detail page's
// neighbourhood emit a `flowchart LR` from ListDependencies edges and draw it
// with the same mermaid pass documents use, so no layout library is added.
//
// Arrows point from what comes first to what follows: blocker → blocked,
// parent → child, origin → discovered item. `related` has no direction and is
// drawn as a plain line. Edge kinds are told apart by line style and label.
import type { IssueEdge, IssueStatus } from "@/api/client.js";

export interface GraphNode {
  id: string;
  title: string;
  status: IssueStatus;
  /** Drawn with a thick border: the issue whose neighbourhood this is. */
  current?: boolean;
}

export interface Flowchart {
  /** The mermaid source. */
  source: string;
  /** Maps a rendered SVG node element id (`flowchart-n3-12`) back to the
   *  issue id, for click-through without mermaid's `click` directive. */
  issueIdOf(svgNodeId: string): string | undefined;
}

const TITLE_MAX = 40;

// Fill and stroke per status, close to the daisyUI badge colours the list
// uses (info, warning, error, success); readable on both themes. The
// current node gets the primary pair on top of its status stroke.
const STATUS_CLASS: Record<IssueStatus, string> = {
  ISSUE_STATUS_UNSPECIFIED: "st_unknown",
  ISSUE_STATUS_OPEN: "st_open",
  ISSUE_STATUS_IN_PROGRESS: "st_in_progress",
  ISSUE_STATUS_BLOCKED: "st_blocked",
  ISSUE_STATUS_CLOSED: "st_closed",
};

const CLASS_DEFS = [
  "classDef st_unknown fill:#e5e7eb,stroke:#6b7280,stroke-width:1.5px,color:#111827",
  "classDef st_open fill:#dbeafe,stroke:#2563eb,stroke-width:1.5px,color:#111827",
  "classDef st_in_progress fill:#fef3c7,stroke:#d97706,stroke-width:1.5px,color:#111827",
  "classDef st_blocked fill:#fee2e2,stroke:#dc2626,stroke-width:1.5px,color:#111827",
  "classDef st_closed fill:#dcfce7,stroke:#16a34a,stroke-width:1.5px,color:#111827",
];

const CURRENT_STYLE = "fill:#c7d2fe,stroke:#4338ca,stroke-width:3px,color:#111827";

/** Escapes text for a quoted mermaid label; mermaid's own entity codes keep
 *  the label out of the parser's way. */
function label(s: string): string {
  return s
    .replace(/#/g, "#35;")
    .replace(/\\/g, "#92;") // a literal backslash-n in a title is not a line break
    .replace(/"/g, "#quot;")
    .replace(/</g, "#lt;")
    .replace(/>/g, "#gt;")
    .replace(/&/g, "#amp;");
}

function shorten(s: string): string {
  return s.length > TITLE_MAX ? `${s.slice(0, TITLE_MAX - 1)}…` : s;
}

function edgeLine(from: string, to: string, type: string): string {
  // `to` is the blocker / parent / origin, so the arrow leaves it.
  switch (type) {
    case "blocks":
      return `${to} -->|blocks| ${from}`;
    case "parent-child":
      return `${to} -.->|child| ${from}`;
    case "discovered-from":
      return `${to} -. discovered .-> ${from}`;
    case "related":
      return `${from} ---|related| ${to}`;
    default:
      return `${to} -->|${label(type)}| ${from}`;
  }
}

/** Builds the flowchart for `nodes`; edges whose ends are not both in `nodes`
 *  are dropped. Node ids are `n<index>`: issue ids carry `.` and `-`, which
 *  mermaid's id grammar does not accept. */
export function flowchart(nodes: GraphNode[], edges: IssueEdge[]): Flowchart {
  const index = new Map<string, string>();
  const lines = ["flowchart LR"];
  nodes.forEach((n, i) => {
    const id = `n${i}`;
    index.set(n.id, id);
    // Rounded node, bold title over the id. Under securityLevel strict the
    // label is sanitized, and <b> / <br/> / <small> survive.
    lines.push(`  ${id}("<b>${label(shorten(n.title))}</b><br/><small>${label(n.id)}</small>"):::${STATUS_CLASS[n.status]}`);
    if (n.current) lines.push(`  style ${id} ${CURRENT_STYLE}`);
  });
  for (const e of edges) {
    const from = index.get(e.fromId);
    const to = index.get(e.toId);
    if (from === undefined || to === undefined) continue;
    lines.push(`  ${edgeLine(from, to, e.type)}`);
  }
  lines.push(...CLASS_DEFS.map((d) => `  ${d}`));

  const back = new Map<string, string>();
  for (const [issueId, id] of index) back.set(id, issueId);
  return {
    source: lines.join("\n"),
    issueIdOf(svgNodeId) {
      // mermaid 11 names a node `<diagram id>-flowchart-<node id>-<n>`; the
      // diagram id prefix is per render, so only the tail is matched.
      const m = /flowchart-(n\d+)-\d+$/.exec(svgNodeId);
      return m ? back.get(m[1]) : undefined;
    },
  };
}
