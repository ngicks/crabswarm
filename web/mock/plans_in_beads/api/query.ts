// The query language of the search bar, GitHub style: qualifiers such as
// `is:open label:chat -label:tui type:epic priority:<2` plus free text
// matched against title and id. liqe parses the text (its Lucene-like grammar
// covers quoting, `-` / NOT, AND / OR and parentheses); this module gives the
// qualifiers their meaning over IssueSummary and offers the suggestions the
// bar shows for the token under the caret.
//
// The query is the filter: the URL's `q` carries it, and the sidebar widgets
// add and remove tokens in it rather than keeping a state of their own.
import { type LiqeQuery, type ParserAst, SyntaxError as LiqeSyntaxError, parse } from "liqe";
import type { IssueStatus, IssueSummary } from "./client.js";

/** What the bar holds when the URL says nothing: GitHub's own default. Here
 *  `is:open` means "not closed", which is also bd's default listing. */
export const DEFAULT_QUERY = "is:open";

const STATUS_WORDS: Record<string, IssueStatus> = {
  open: "ISSUE_STATUS_OPEN",
  in_progress: "ISSUE_STATUS_IN_PROGRESS",
  blocked: "ISSUE_STATUS_BLOCKED",
  closed: "ISSUE_STATUS_CLOSED",
};

/** One qualifier the bar understands. `values` lists what can follow the
 *  colon, for suggestions; `labels` and `parents` come from the source. */
export interface Qualifier {
  name: string;
  help: string;
  values(ctx: SuggestContext): string[];
}

export interface SuggestContext {
  labels: string[];
  ids: string[];
}

export const QUALIFIERS: Qualifier[] = [
  { name: "is", help: "open (not closed), closed, plan", values: () => ["open", "closed", "plan"] },
  { name: "status", help: "exact bd status", values: () => Object.keys(STATUS_WORDS) },
  { name: "label", help: "has the label; -label: excludes", values: (c) => c.labels },
  { name: "type", help: "task, epic, bug, …", values: () => ["task", "epic", "bug", "feature", "chore"] },
  { name: "parent", help: "child of this issue", values: (c) => c.ids },
  { name: "priority", help: "priority:1, priority:<2, priority:>=2", values: () => ["0", "1", "2", "3", "4"] },
];

export interface ParsedQuery {
  ast: LiqeQuery | null;
  /** Parser message when the text does not parse; the previous rows stay. */
  error: string;
  /** Qualifiers the text uses that this bar does not know. They match nothing. */
  unknown: string[];
}

const parseCache = new Map<string, ParsedQuery>();

export function parseQuery(text: string): ParsedQuery {
  const cached = parseCache.get(text);
  if (cached) return cached;
  let out: ParsedQuery;
  try {
    const ast = text.trim() === "" ? null : parse(text);
    const unknown: string[] = [];
    if (ast) collectUnknown(ast, unknown);
    out = { ast, error: "", unknown };
  } catch (e) {
    // liqe raises its SyntaxError for most mistakes, and a plain Error from
    // the grammar for the rest (an unterminated quote, for one).
    const message = e instanceof LiqeSyntaxError ? `${e.message} (column ${e.column})` : e instanceof Error ? e.message : String(e);
    out = { ast: null, error: message, unknown: [] };
  }
  parseCache.set(text, out);
  return out;
}

function collectUnknown(ast: ParserAst, out: string[]): void {
  switch (ast.type) {
    case "Tag": {
      const field = ast.field;
      if (field.type === "Field" && !QUALIFIERS.some((q) => q.name === field.name) && !out.includes(field.name)) {
        out.push(field.name);
      }
      return;
    }
    case "LogicalExpression":
      collectUnknown(ast.left, out);
      collectUnknown(ast.right, out);
      return;
    case "UnaryOperator":
      collectUnknown(ast.operand, out);
      return;
    case "ParenthesizedExpression":
      collectUnknown(ast.expression, out);
      return;
    case "EmptyExpression":
      return;
  }
}

/** matches evaluates the parsed query against one issue. An empty query
 *  matches everything. */
export function matches(ast: LiqeQuery | null, s: IssueSummary): boolean {
  if (!ast) return true;
  return evaluate(ast, s);
}

function evaluate(ast: ParserAst, s: IssueSummary): boolean {
  switch (ast.type) {
    case "EmptyExpression":
      return true;
    case "LogicalExpression":
      return ast.operator.operator === "OR"
        ? evaluate(ast.left, s) || evaluate(ast.right, s)
        : evaluate(ast.left, s) && evaluate(ast.right, s);
    case "UnaryOperator":
      return !evaluate(ast.operand, s);
    case "ParenthesizedExpression":
      return evaluate(ast.expression, s);
    case "Tag":
      return evaluateTag(ast, s);
  }
}

function evaluateTag(tag: Extract<ParserAst, { type: "Tag" }>, s: IssueSummary): boolean {
  const expr = tag.expression;
  // `label:` with nothing after the colon is a token still being typed.
  if (expr.type === "EmptyExpression") return true;
  if (expr.type === "RegexExpression" || expr.type === "RangeExpression") {
    return tag.field.type === "Field" && tag.field.name === "priority" && expr.type === "RangeExpression"
      ? inRange(s.priority, expr.range)
      : false;
  }
  const value = String(expr.value).toLowerCase();

  if (tag.field.type === "ImplicitField") {
    return s.title.toLowerCase().includes(value) || s.id.toLowerCase().includes(value);
  }
  switch (tag.field.name) {
    case "is":
      if (value === "open") return s.status !== "ISSUE_STATUS_CLOSED";
      if (value === "closed") return s.status === "ISSUE_STATUS_CLOSED";
      if (value === "plan") return s.labels.includes("plan");
      return false;
    case "status":
      return STATUS_WORDS[value] === s.status;
    case "label":
      return s.labels.some((l) => l.toLowerCase() === value);
    case "type":
      return s.issueType.toLowerCase() === value;
    case "parent":
      return s.parentId.toLowerCase() === value;
    case "priority":
      return compare(s.priority, tag.operator.operator, Number(expr.value));
    default:
      return false;
  }
}

function compare(actual: number, op: string, wanted: number): boolean {
  if (Number.isNaN(wanted)) return false;
  switch (op) {
    case ":<":
      return actual < wanted;
    case ":<=":
      return actual <= wanted;
    case ":>":
      return actual > wanted;
    case ":>=":
      return actual >= wanted;
    default:
      return actual === wanted;
  }
}

function inRange(n: number, r: { min: number; max: number; minInclusive: boolean; maxInclusive: boolean }): boolean {
  const lo = r.minInclusive ? n >= r.min : n > r.min;
  const hi = r.maxInclusive ? n <= r.max : n < r.max;
  return lo && hi;
}

// --- tokens: the sidebar widgets and the suggestions edit the text ---------

export interface Token {
  text: string;
  start: number;
  end: number;
}

/** Whitespace-separated tokens, quotes kept together. Good enough for the
 *  widgets' `field:value` edits; liqe does the real parsing. */
export function tokensOf(text: string): Token[] {
  const out: Token[] = [];
  const re = /"[^"]*"?|'[^']*'?|\S+/g;
  for (const m of text.matchAll(re)) {
    out.push({ text: m[0], start: m.index, end: m.index + m[0].length });
  }
  return out;
}

/** The token the caret is in or right after; a caret on whitespace starts a
 *  new empty token there. */
export function tokenAt(text: string, caret: number): Token {
  for (const t of tokensOf(text)) {
    if (caret >= t.start && caret <= t.end) return t;
  }
  return { text: "", start: caret, end: caret };
}

export function replaceToken(text: string, token: Token, replacement: string): string {
  return text.slice(0, token.start) + replacement + text.slice(token.end);
}

function tagToken(field: string, value: string): string {
  return /[\s"']/.test(value) ? `${field}:"${value.replace(/"/g, '\\"')}"` : `${field}:${value}`;
}

/** Whether `field:value` (not negated) is a token of the text. */
export function hasTag(text: string, field: string, value: string): boolean {
  const want = tagToken(field, value);
  return tokensOf(text).some((t) => t.text === want);
}

/** Adds `field:value` when absent, removes it when present. `dropTokens`
 *  are exact tokens removed alongside an addition (a `status:` pick
 *  replaces `is:open`, which would otherwise AND with it; `is:plan` stays). */
export function toggleTag(text: string, field: string, value: string, dropTokens: string[] = []): string {
  const want = tagToken(field, value);
  const tokens = tokensOf(text);
  const kept = tokens.filter((t) => t.text !== want);
  if (kept.length !== tokens.length) return kept.map((t) => t.text).join(" ");
  const rest = kept.filter((t) => !dropTokens.includes(t.text));
  return [...rest.map((t) => t.text), want].join(" ");
}

/** Values of every `field:` tag in the text, for the widgets to reflect. */
export function tagValues(text: string, field: string): string[] {
  const prefix = `${field}:`;
  return tokensOf(text)
    .filter((t) => t.text.startsWith(prefix))
    .map((t) => t.text.slice(prefix.length).replace(/^"(.*)"$/, "$1"));
}

// --- suggestions -----------------------------------------------------------

export interface Suggestion {
  /** What replaces the current token when picked. */
  insert: string;
  label: string;
  help: string;
}

/** Suggestions for the token being typed: qualifier names before the colon,
 *  their values after it. A token with no colon also offers itself as free
 *  text so the list never claims the input is wrong. */
export function suggest(token: string, ctx: SuggestContext): Suggestion[] {
  const negated = token.startsWith("-");
  const body = negated ? token.slice(1) : token;
  const sign = negated ? "-" : "";
  const colon = body.indexOf(":");
  if (colon < 0) {
    const needle = body.toLowerCase();
    return QUALIFIERS.filter((q) => q.name.startsWith(needle)).map((q) => ({
      insert: `${sign}${q.name}:`,
      label: `${sign}${q.name}:`,
      help: q.help,
    }));
  }
  const field = body.slice(0, colon);
  const typed = body.slice(colon + 1).toLowerCase();
  const q = QUALIFIERS.find((x) => x.name === field);
  if (!q) return [];
  return q
    .values(ctx)
    .filter((v) => v.toLowerCase().includes(typed))
    .slice(0, 12)
    .map((v) => ({ insert: `${sign}${tagToken(field, v)}`, label: `${sign}${field}:${v}`, help: "" }));
}
