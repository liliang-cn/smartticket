import { useState } from "react";
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
import { Button } from "@/components/ui/button";
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
  const cancel = useCancelJob();
  const remove = useDeleteJob();
  const download = useDownloadJob();
  const canCancel = job.status === "pending" || job.status === "running";
  const canDownload = job.type === "export" && job.status === "completed";

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
          {download.isPending ? "Downloading…" : "Download"}
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
              Cancel job
            </DropdownMenuItem>
          )}
          <DropdownMenuItem
            disabled={remove.isPending}
            className="text-destructive focus:text-destructive"
            onSelect={() => remove.mutate(job.id)}
          >
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export function DataJobsPage() {
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
    <div className="mx-auto max-w-6xl">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-mono text-xs uppercase tracking-[0.25em] text-muted-foreground">
            data
          </div>
          <h1 className="mt-1 text-3xl">Import / Export</h1>
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
            <StatCard label="Total jobs" value={totalJobs ?? "—"} />
            <StatCard label="Completed" value={completed} />
            <StatCard label="Running" value={running} />
            <StatCard label="Failed" value={failed} />
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
            <SelectValue placeholder="Type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All types</SelectItem>
            {TYPE_OPTIONS.map((t) => (
              <SelectItem key={t} value={t} className="capitalize">
                {t}
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
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>All statuses</SelectItem>
            {STATUS_OPTIONS.map((s) => (
              <SelectItem key={s} value={s} className="capitalize">
                {s}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border text-left font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
              <th className="px-4 py-3 font-medium">Type</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Progress</th>
              <th className="px-4 py-3 font-medium">Records</th>
              <th className="px-4 py-3 font-medium">Created</th>
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
                        {j.type}
                      </span>
                    </div>
                    <div className="mt-0.5 font-mono text-[11px] text-muted-foreground">
                      {j.type === "export"
                        ? j.target_format || "—"
                        : j.source_format || "—"}
                    </div>
                  </td>
                  <td className="px-4 py-3.5">
                    <Badge tone={STATUS_TONE[j.status] ?? "slate"}>
                      {j.status}
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
                        · {j.failed_records} failed
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
                    No import/export jobs match these filters.
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
            {data.total} jobs · page {data.page}/{data.total_pages}
            {isFetching && " · syncing…"}
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={filters.page <= 1}
              onClick={() => setFilters((f) => ({ ...f, page: f.page - 1 }))}
            >
              <ChevronLeft /> Prev
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={data.page >= data.total_pages}
              onClick={() => setFilters((f) => ({ ...f, page: f.page + 1 }))}
            >
              Next <ChevronRight />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
