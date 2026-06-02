import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  ChevronLeft,
  ChevronRight,
  Database,
  Download,
  MoreHorizontal,
  Layers,
} from "lucide-react";
import {
  useJobs,
  useJobStats,
  useCancelJob,
  useDeleteJob,
  useDownloadJob,
  type JobFilters,
  type JobStatus,
  type JobType,
  type Job,
} from "@/features/data/api";
import { ExportJobDialog } from "@/features/data/export-job-dialog";
import { toast } from "sonner";
import { apiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/misc";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { relativeTime } from "@/lib/utils";

const ALL = "__all__";

const STATUS_TONE: Record<JobStatus, "amber" | "blue" | "green" | "red" | "slate"> = {
  pending: "amber",
  running: "blue",
  completed: "green",
  failed: "red",
  cancelled: "slate",
};

const TYPE_OPTIONS: JobType[] = ["import", "export"];
const STATUS_OPTIONS: JobStatus[] = [
  "pending",
  "running",
  "completed",
  "failed",
  "cancelled",
];

function ProgressBar({ value }: { value: number }) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full bg-primary transition-all"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="font-mono text-[11px] tabular-nums text-muted-foreground">
        {pct}%
      </span>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <Card className="relative overflow-hidden p-5">
      <div className="font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
        {label}
      </div>
      <div className="mt-3 font-display text-3xl font-bold tabular-nums">
        {value}
      </div>
    </Card>
  );
}

function RowActions({ job }: { job: Job }) {
  const { t } = useTranslation("data");
  const cancel = useCancelJob();
  const remove = useDeleteJob();
  const download = useDownloadJob();
  const [toDelete, setToDelete] = useState<{ id: number; label: string } | null>(
    null
  );
  const canCancel = job.status === "pending" || job.status === "running";
  // A job is a downloadable export when it completed and produced a result file.
  // (job.type holds the exported entity — e.g. "tickets" — not "export"; imports
  // never set a file_path, so this distinguishes exports reliably.)
  const canDownload = job.status === "completed" && !!job.file_path;

  async function confirmDelete() {
    if (!toDelete) return;
    try {
      await remove.mutateAsync(toDelete.id);
      toast.success(t("toasts.job_deleted"));
      setToDelete(null);
    } catch (err) {
      toast.error(apiError(err, t("toasts.delete_failed")));
    }
  }

  return (
    <div className="flex items-center justify-end gap-1">
      {canDownload && (
        <Button
          variant="ghost"
          size="sm"
          disabled={download.isPending}
          onClick={(e) => {
            e.stopPropagation();
            download.mutate({ id: job.id, file_path: job.file_path });
          }}
        >
          <Download className="size-4" />
          {download.isPending ? t("actions.downloading") : t("actions.download")}
        </Button>
      )}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            onClick={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
          {canCancel && (
            <DropdownMenuItem
              disabled={cancel.isPending}
              onSelect={() => cancel.mutate(job.id)}
            >
              {t("actions.cancel_job")}
            </DropdownMenuItem>
          )}
          <DropdownMenuItem
            disabled={remove.isPending}
            className="text-destructive focus:text-destructive"
            onSelect={(e) => {
              e.preventDefault();
              setToDelete({ id: job.id, label: job.type });
            }}
          >
            {t("actions.delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <ConfirmDialog
        open={!!toDelete}
        onOpenChange={(o) => !o && setToDelete(null)}
        title={t("confirm_delete.title")}
        description={
          toDelete
            ? t("confirm_delete.description", {
                id: toDelete.id,
                label: toDelete.label,
              })
            : undefined
        }
        pending={remove.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}

export function DataJobsPage() {
  const { t } = useTranslation("data");
  const [filters, setFilters] = useState<JobFilters>({
    page: 1,
    page_size: 15,
  });
  const { data, isLoading, isFetching } = useJobs(filters);
  const { data: stats, isLoading: statsLoading } = useJobStats();

  const set = (patch: Partial<JobFilters>) =>
    setFilters((f) => ({ ...f, page: 1, ...patch }));

  // Defensive reads — the stats endpoint returns a free-form map.
  const totalJobs =
    typeof stats?.total_jobs === "number" ? stats.total_jobs : undefined;
  const statusBreakdown = stats?.status_breakdown ?? {};
  const completed = statusBreakdown["completed"] ?? 0;
  const running = statusBreakdown["running"] ?? 0;
  const failed = statusBreakdown["failed"] ?? 0;

  return (
    <div className="w-full">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            {t("page.section_label")}
          </div>
          <h1 className="mt-1 text-3xl">{t("page.title")}</h1>
        </div>
        <ExportJobDialog />
      </div>

      {/* Stats */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {statsLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <Card key={i} className="p-5">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="mt-3 h-8 w-16" />
            </Card>
          ))
        ) : (
          <>
            <StatCard label={t("stats.total_jobs")} value={totalJobs ?? "—"} />
            <StatCard label={t("stats.completed")} value={completed} />
            <StatCard label={t("stats.running")} value={running} />
            <StatCard label={t("stats.failed")} value={failed} />
          </>
        )}
      </div>

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Select
          value={filters.type ?? ALL}
          onValueChange={(v) =>
            set({ type: v === ALL ? undefined : (v as JobType) })
          }
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder={t("filters.type_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("filters.all_types")}</SelectItem>
            {TYPE_OPTIONS.map((type) => (
              <SelectItem key={type} value={type}>
                {t(`job_type.${type}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={filters.status ?? ALL}
          onValueChange={(v) =>
            set({ status: v === ALL ? undefined : (v as JobStatus) })
          }
        >
          <SelectTrigger className="w-40">
            <SelectValue placeholder={t("filters.status_placeholder")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{t("filters.all_statuses")}</SelectItem>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s} value={s}>
                {t(`status.${s}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">{t("table.col_type")}</th>
              <th className="px-4 py-3 font-medium">{t("table.col_status")}</th>
              <th className="px-4 py-3 font-medium">{t("table.col_progress")}</th>
              <th className="px-4 py-3 font-medium">{t("table.col_records")}</th>
              <th className="px-4 py-3 font-medium">{t("table.col_created")}</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-b border-border/60">
                  {Array.from({ length: 6 }).map((__, j) => (
                    <td key={j} className="px-4 py-3.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data && data.items.length > 0 ? (
              data.items.map((j) => (
                <tr
                  key={j.id}
                  className="border-b border-border/60 transition-colors last:border-0 hover:bg-accent/50"
                >
                  <td className="px-4 py-3.5">
                    <div className="flex items-center gap-2">
                      <Layers className="size-4 text-muted-foreground/70" />
                      <span className="font-medium capitalize text-foreground">
                        {t(`job_type.${j.type}`, { defaultValue: j.type })}
                      </span>
                    </div>
                    <div className="mt-0.5 font-mono text-[11px] text-muted-foreground">
                      {j.target_format || j.source_format || "—"}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={STATUS_TONE[j.status] ?? "slate"}>
                      {t(`status.${j.status}`, { defaultValue: j.status })}
                    </Badge>
                    {j.status === "failed" && j.error && (
                      <div
                        className="mt-1 max-w-48 truncate text-[11px] text-destructive"
                        title={j.error}
                      >
                        {j.error}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-3.5">
                    <ProgressBar value={j.progress} />
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {j.processed_records}/{j.total_records}
                    {j.failed_records > 0 && (
                      <span className="text-destructive">
                        {" "}
                        · {t("records.failed_count", { count: j.failed_records })}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                    {relativeTime(j.created_at)}
                  </td>
                  <td className="px-2 py-3.5 text-right">
                    <RowActions job={j} />
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-4 py-16 text-center">
                  <Database className="mx-auto size-8 text-muted-foreground/40" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    {t("table.empty_heading")}
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
            {t("pagination.summary", {
              total: data.total,
              page: data.page,
              totalPages: data.total_pages,
            })}
            {isFetching && t("pagination.syncing")}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> {t("pagination.prev")}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              {t("pagination.next")} <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
