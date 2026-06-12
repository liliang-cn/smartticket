import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import type { TicketPriority, TicketSeverity, TicketStatus } from "@/lib/types";

const STATUS_TONE: Record<TicketStatus, Parameters<typeof Badge>[0]["tone"]> = {
  open: "amber",
  in_progress: "blue",
  resolved: "green",
  closed: "slate",
  cancelled: "neutral",
  merged: "slate",
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
  const { t } = useTranslation("common");
  return (
    <Badge tone={STATUS_TONE[status] ?? "neutral"} className="uppercase">
      <span className="inline-block size-1.5 rounded-full bg-current" />
      {t(`enums.ticket_status.${status}`)}
    </Badge>
  );
}

export function PriorityBadge({ priority }: { priority: TicketPriority }) {
  const { t } = useTranslation("common");
  return (
    <Badge tone={PRIORITY_TONE[priority] ?? "neutral"} className="uppercase">
      {t(`enums.priority.${priority}`)}
    </Badge>
  );
}

const SEVERITY_TONE: Record<TicketSeverity, Parameters<typeof Badge>[0]["tone"]> = {
  trivial: "slate",
  minor: "blue",
  major: "amber",
  critical: "red",
};

export function SeverityBadge({ severity }: { severity: TicketSeverity }) {
  const { t } = useTranslation("common");
  return (
    <Badge tone={SEVERITY_TONE[severity] ?? "neutral"} className="uppercase">
      {t(`enums.severity.${severity}`)}
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
