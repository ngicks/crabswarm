// How wire values are spelled on screen: protobuf timestamps and the issue
// status enum.
import { type Timestamp, timestampMs } from "@bufbuild/protobuf/wkt";
import { IssueStatus } from "@/api/gen/ngicks/crabswarm/issues/v1/issues_service_pb.js";

/** Short "2026-09-04 14:41" stamp. An absent timestamp — bd omits the field
 *  when the issue does not carry it — reads as nothing at all. */
export function shortTime(ts: Timestamp | undefined): string {
  return ts ? shortTimeMs(timestampMs(ts)) : "";
}

/** The same stamp from Unix milliseconds, for values already reduced to a
 *  number (the labels page aggregates over timestamps before spelling them). */
export function shortTimeMs(ms: number): string {
  const d = new Date(ms);
  if (ms === 0 || Number.isNaN(d.getTime())) return "";
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

/** Unix milliseconds, 0 when unset: the ordering key wherever timestamps are
 *  compared, so an issue bd left without one sorts last rather than throwing. */
export function timeValue(ts: Timestamp | undefined): number {
  return ts ? timestampMs(ts) : 0;
}

const STATUS_LABELS: Record<IssueStatus, string> = {
  [IssueStatus.UNSPECIFIED]: "unknown",
  [IssueStatus.OPEN]: "open",
  [IssueStatus.IN_PROGRESS]: "in_progress",
  [IssueStatus.BLOCKED]: "blocked",
  [IssueStatus.CLOSED]: "closed",
  [IssueStatus.DEFERRED]: "deferred",
};

const STATUS_BADGES: Record<IssueStatus, string> = {
  [IssueStatus.UNSPECIFIED]: "badge-ghost",
  [IssueStatus.OPEN]: "badge-info",
  [IssueStatus.IN_PROGRESS]: "badge-warning",
  [IssueStatus.BLOCKED]: "badge-error",
  [IssueStatus.CLOSED]: "badge-success",
  [IssueStatus.DEFERRED]: "badge-neutral",
};

export function statusLabel(s: IssueStatus): string {
  return STATUS_LABELS[s] ?? "unknown";
}

export function statusBadgeClass(s: IssueStatus): string {
  return STATUS_BADGES[s] ?? "badge-ghost";
}
