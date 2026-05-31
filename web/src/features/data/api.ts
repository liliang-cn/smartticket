import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

// ---------------------------------------------------------------------------
// Domain types (local to the data import/export feature).
// ---------------------------------------------------------------------------

export type JobType = "import" | "export";
export type JobStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

/** A single import/export job, mirroring the backend JobResponse payload. */
export interface Job {
  id: number;
  type: JobType;
  status: JobStatus;
  progress: number;
  total_records: number;
  processed_records: number;
  failed_records: number;
  source_format: string;
  target_format: string;
  file_path: string;
  error?: string;
  started_at?: string | null;
  completed_at?: string | null;
  created_at: string;
}

/** Export entity types the backend accepts. */
export type ExportEntity =
  | "tickets"
  | "knowledge_articles"
  | "users"
  | "products"
  | "services"
  | "complete";

/** Target formats the backend accepts for an export. */
export type ExportFormat = "csv" | "json" | "xml" | "markdown" | "sqlite";

export interface JobFilters {
  page: number;
  page_size: number;
  type?: JobType;
  status?: JobStatus;
}

export interface JobPage {
  items: Job[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

/**
 * Loosely-typed job statistics. The backend returns a free-form map
 * (status_breakdown, type_breakdown, total_jobs, recent_activity, …), so the
 * page reads these defensively rather than relying on a fixed shape.
 */
export interface JobStats {
  total_jobs?: number;
  status_breakdown?: Record<string, number>;
  type_breakdown?: Record<string, number>;
  [key: string]: unknown;
}

export function useJobs(filters: JobFilters) {
  return useQuery({
    queryKey: ["data-jobs", filters],
    queryFn: async (): Promise<JobPage> => {
      const res = await api.get("/data/jobs", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          type: filters.type || undefined,
          status: filters.status || undefined,
        },
      });
      // The list endpoint returns { success, data: Job[], meta: {...} }.
      const body = res.data as {
        data?: Job[];
        meta?: {
          total?: number;
          page?: number;
          page_size?: number;
          total_pages?: number;
        };
      };
      const meta = body.meta ?? {};
      return {
        items: body.data ?? [],
        total: meta.total ?? 0,
        page: meta.page ?? filters.page,
        page_size: meta.page_size ?? filters.page_size,
        total_pages: meta.total_pages ?? 1,
      };
    },
    placeholderData: (prev) => prev,
  });
}

export function useJob(id: number | undefined) {
  return useQuery({
    queryKey: ["data-job", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/data/jobs/${id}`);
      return unwrap<Job>(res.data);
    },
  });
}

export function useJobStats() {
  return useQuery({
    queryKey: ["data-job-stats"],
    queryFn: async () => {
      const res = await api.get("/data/jobs/stats");
      return unwrap<JobStats>(res.data) ?? {};
    },
  });
}

export interface CreateExportJobInput {
  /** Entity to export (maps to the backend's `type` field). */
  type: ExportEntity;
  target_format: ExportFormat;
}

export function useCreateExportJob() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateExportJobInput) => {
      const res = await api.post("/data/jobs/export", input);
      return unwrap<Job>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["data-jobs"] });
      qc.invalidateQueries({ queryKey: ["data-job-stats"] });
    },
  });
}

export function useCancelJob() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.post(`/data/jobs/${id}/cancel`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["data-jobs"] });
      qc.invalidateQueries({ queryKey: ["data-job-stats"] });
    },
  });
}

export function useDeleteJob() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/data/jobs/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["data-jobs"] });
      qc.invalidateQueries({ queryKey: ["data-job-stats"] });
    },
  });
}

/**
 * Download an export job's result file. The download endpoint is JWT-protected,
 * so a plain anchor (`<a href>`) cannot attach the Authorization header — we
 * fetch the response as a blob through the configured axios client (which
 * injects the token) and trigger a client-side save.
 */
export function useDownloadJob() {
  return useMutation({
    mutationFn: async (job: Pick<Job, "id" | "file_path">) => {
      const res = await api.get(`/data/jobs/${job.id}/download`, {
        responseType: "blob",
      });
      const blob = res.data as Blob;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download =
        job.file_path?.split("/").pop() || `export-job-${job.id}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    },
  });
}
