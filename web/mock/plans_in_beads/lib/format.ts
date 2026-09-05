// How wire values are spelled on screen: timestamps and the status enum.
import type { IssueStatus } from "@/api/client.js";

/** Short "2026-09-04 14:41" stamp; the fixture carries RFC 3339. */
export function shortTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

const STATUS_LABELS: Record<IssueStatus, string> = {
  ISSUE_STATUS_UNSPECIFIED: "unknown",
  ISSUE_STATUS_OPEN: "open",
  ISSUE_STATUS_IN_PROGRESS: "in_progress",
  ISSUE_STATUS_BLOCKED: "blocked",
  ISSUE_STATUS_CLOSED: "closed",
};

const STATUS_BADGES: Record<IssueStatus, string> = {
  ISSUE_STATUS_UNSPECIFIED: "badge-ghost",
  ISSUE_STATUS_OPEN: "badge-info",
  ISSUE_STATUS_IN_PROGRESS: "badge-warning",
  ISSUE_STATUS_BLOCKED: "badge-error",
  ISSUE_STATUS_CLOSED: "badge-success",
};

export function statusLabel(s: IssueStatus): string {
  return STATUS_LABELS[s] ?? s;
}

export function statusBadgeClass(s: IssueStatus): string {
  return STATUS_BADGES[s] ?? "badge-ghost";
}
