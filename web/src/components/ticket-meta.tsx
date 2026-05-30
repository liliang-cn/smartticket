import { Badge } from "@/components/ui/badge";
import type { TicketPriority, TicketStatus } from "@/lib/types";

const STATUS_TONE: Record<TicketStatus, Parameters<typeof Badge>[0]["tone"]> = {
  open: "amber",
  in_progress: "blue",
  resolved: "green",
  closed: "slate",
  cancelled: "neutral",
};

const STATUS_LABEL: Record<TicketStatus, string> = {
  open: "Open",
  in_progress: "In progress",
  resolved: "Resolved",
  closed: "Closed",
  cancelled: "Cancelled",
};

const PRIORITY_TONE: Record<
  TicketPriority,
  Parameters<typeof Badge>[0]["tone"]
> = {
  low: "slate",
  medium: "blue",
  high: "amber",
  critical: "red",
};

export function StatusBadge({ status }: { status: TicketStatus }) {
  return (
    <Badge tone={STATUS_TONE[status] ?? "neutral"} className="uppercase">
      <span className="inline-block size-1.5 rounded-full bg-current" />
      {STATUS_LABEL[status] ?? status}
    </Badge>
  );
}

export function PriorityBadge({ priority }: { priority: TicketPriority }) {
  return (
    <Badge tone={PRIORITY_TONE[priority] ?? "neutral"} className="uppercase">
      {priority}
    </Badge>
  );
}

export const STATUS_OPTIONS: TicketStatus[] = [
  "open",
  "in_progress",
  "resolved",
  "closed",
  "cancelled",
];
export const PRIORITY_OPTIONS: TicketPriority[] = [
  "low",
  "medium",
  "high",
  "critical",
];
