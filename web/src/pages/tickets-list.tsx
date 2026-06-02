import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Search, ChevronLeft, ChevronRight, Inbox } from "lucide-react";
import { useTickets, type TicketFilters } from "@/features/tickets/api";
import { CreateTicketDialog } from "@/features/tickets/create-ticket-dialog";
import {
  PriorityBadge,
  StatusBadge,
  STATUS_OPTIONS,
  PRIORITY_OPTIONS,
} from "@/components/ticket-meta";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/misc";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { relativeTime } from "@/lib/utils";
import { useAuth } from "@/lib/auth";

const ALL = "__all__";

export function TicketsListPage() {
  const navigate = useNavigate();
  const { t } = useTranslation("tickets");
  const { t: tCommon } = useTranslation("common");
  const { user } = useAuth();
  // Customers see only their own org's tickets, so the customer column is
  // redundant for them; show it to team users who triage across customers.
  const isTeam = user?.role !== "customer";
  const colCount = isTeam ? 6 : 5;
  const [filters, setFilters] = useState<TicketFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useTickets(filters);

  const set = (patch: Partial<TicketFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("list.queue_label")}
          </div>
          <h1 className="mt-1 text-3xl">{t("list.heading")}</h1>
        </div>
        <CreateTicketDialog />
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative min-w-56 flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t("list.search_placeholder")}
            value={filters.search ?? ""}
            onChange={(e) => set({ search: e.target.value })}
          />
        </div>
        <Select
          value={filters.status ?? ALL}
          onValueChange={(v) => set({ status: v === ALL ? undefined : v })}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder={t("list.filter_status_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("list.filter_status_all")}</SelectItem>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s} value={s}>
                {tCommon(`enums.ticket_status.${s}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={filters.priority ?? ALL}
          onValueChange={(v) => set({ priority: v === ALL ? undefined : v })}
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder={t("list.filter_priority_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("list.filter_priority_all")}</SelectItem>
            {PRIORITY_OPTIONS.map((p) => (
              <SelectItem key={p} value={p}>
                {tCommon(`enums.priority.${p}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("list.col_ticket")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_status")}</th>
              <th className="px-4 py-3 font-medium">{t("list.col_priority")}</th>
              {isTeam && <th className="px-4 py-3 font-medium">{t("list.col_customer")}</th>}
              <th className="px-4 py-3 font-medium">{t("list.col_requester")}</th>
              <th className="px-4 py-3 text-right font-medium">{t("list.col_updated")}</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: colCount }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data && data.items.length > 0 ? (
              data.items.map((ticket) => (
                <tr
                  key={ticket.id}
                  onClick={() => navigate(`/tickets/${ticket.id}`)}
                  className="group cursor-pointer border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs text-primary/80">
                        {ticket.ticket_number}
                      </span>
                    </div>
                    <div className="mt-0.5 font-medium text-foreground group-hover:text-primary">
                      {ticket.title}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    <StatusBadge status={ticket.status} />
                  </td>
                  <td className="px-4 py-3.5">
                    <PriorityBadge priority={ticket.priority} />
                  </td>
                  {isTeam && (
                    <td className="px-4 py-3.5 text-muted-foreground">
                      {ticket.customer_name || "—"}
                    </td>
                  )}
                  <td className="px-4 py-3.5 text-muted-foreground">
                    {ticket.requester_name || "—"}
                  </td>
                  <td className="px-4 py-3.5 text-right font-mono text-xs text-muted-foreground">
                    {relativeTime(ticket.updated_at)}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={colCount} className="px-4 py-16 text-center">
                  <Inbox className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("list.empty")}
                  </p>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Pagination */}
      {data && data.total_pages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="font-mono text-xs text-muted-foreground">
            {t("list.pagination_info", { total: data.total, page: data.page, totalPages: data.total_pages })}
            {isFetching && t("list.pagination_syncing")}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> {t("list.btn_prev")}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              {t("list.btn_next")} <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
