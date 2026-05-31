import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

// ── Domain types ────────────────────────────────────────────────────────────

export interface SLATemplate {
  id: number;
  name: string;
  description: string;
  is_default: boolean;
  is_active: boolean;
  priority_levels: string[];
  severity_levels: string[];
  response_times: Record<string, number>;
  resolution_times: Record<string, number>;
  business_hours: Record<string, unknown>;
  holidays: string[];
  configuration: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface SLARule {
  id: number;
  template_id: number;
  priority: string;
  severity: string;
  /** Response target in minutes. */
  response_time: number;
  /** Resolution target in minutes. */
  resolution_time: number;
  business_only: boolean;
  product_id?: number | null;
  service_id?: number | null;
  conditions: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// ── Shared list helpers ───────────────────────────────────────────────────────

export interface SLAFilters {
  page: number;
  page_size: number;
  search?: string;
}

interface ListBody<T> {
  data?: T[];
  meta?: {
    total?: number;
    page?: number;
    page_size?: number;
    total_pages?: number;
  };
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

function toPage<T>(body: ListBody<T>, filters: SLAFilters): Page<T> {
  const meta = body.meta ?? {};
  return {
    items: body.data ?? [],
    total: meta.total ?? 0,
    page: meta.page ?? filters.page,
    page_size: meta.page_size ?? filters.page_size,
    total_pages: meta.total_pages ?? 1,
  };
}

// ── SLA Templates ─────────────────────────────────────────────────────────────

export function useSLATemplates(filters: SLAFilters) {
  return useQuery({
    queryKey: ["sla-templates", filters],
    queryFn: async (): Promise<Page<SLATemplate>> => {
      const res = await api.get("/admin/sla-templates", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
        },
      });
      // The list endpoint returns { success, data: SLATemplate[], meta: {...} }.
      return toPage(res.data as ListBody<SLATemplate>, filters);
    },
    placeholderData: (prev) => prev,
  });
}

export function useSLATemplate(id: number | undefined) {
  return useQuery({
    queryKey: ["sla-template", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/admin/sla-templates/${id}`);
      return unwrap<SLATemplate>(res.data);
    },
  });
}

export interface CreateSLATemplateInput {
  name: string;
  description?: string;
  is_default?: boolean;
  is_active?: boolean;
}

export function useCreateSLATemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateSLATemplateInput) => {
      const res = await api.post("/admin/sla-templates", input);
      return unwrap<SLATemplate>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-templates"] });
    },
  });
}

export function useUpdateSLATemplate(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<CreateSLATemplateInput>) => {
      const res = await api.put(`/admin/sla-templates/${id}`, patch);
      return unwrap<SLATemplate>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-template", id] });
      qc.invalidateQueries({ queryKey: ["sla-templates"] });
    },
  });
}

export function useDeleteSLATemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/sla-templates/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-templates"] });
    },
  });
}

export function useSetDefaultSLATemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.post(`/admin/sla-templates/${id}/set-default`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-templates"] });
    },
  });
}

export function useSetSLATemplateActive() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, active }: { id: number; active: boolean }) => {
      await api.post(
        `/admin/sla-templates/${id}/${active ? "activate" : "deactivate"}`
      );
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-templates"] });
    },
  });
}

export function useActivateSLATemplate() {
  const m = useSetSLATemplateActive();
  return {
    ...m,
    mutate: (id: number) => m.mutate({ id, active: true }),
    mutateAsync: (id: number) => m.mutateAsync({ id, active: true }),
  };
}

export function useDeactivateSLATemplate() {
  const m = useSetSLATemplateActive();
  return {
    ...m,
    mutate: (id: number) => m.mutate({ id, active: false }),
    mutateAsync: (id: number) => m.mutateAsync({ id, active: false }),
  };
}

// ── SLA Rules ─────────────────────────────────────────────────────────────────

export function useSLARules(filters: SLAFilters) {
  return useQuery({
    queryKey: ["sla-rules", filters],
    queryFn: async (): Promise<Page<SLARule>> => {
      const res = await api.get("/admin/sla-rules", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
        },
      });
      // The list endpoint returns { success, data: SLARule[], meta: {...} }.
      return toPage(res.data as ListBody<SLARule>, filters);
    },
    placeholderData: (prev) => prev,
  });
}

export function useSLARule(id: number | undefined) {
  return useQuery({
    queryKey: ["sla-rule", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/admin/sla-rules/${id}`);
      return unwrap<SLARule>(res.data);
    },
  });
}

export interface CreateSLARuleInput {
  template_id: number;
  priority?: string;
  severity?: string;
  response_time?: number;
  resolution_time?: number;
  business_only?: boolean;
  product_id?: number | null;
  service_id?: number | null;
  conditions?: string;
  is_active?: boolean;
}

export function useCreateSLARule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateSLARuleInput) => {
      const res = await api.post("/admin/sla-rules", input);
      return unwrap<SLARule>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-rules"] });
    },
  });
}

export function useUpdateSLARule(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<CreateSLARuleInput>) => {
      const res = await api.put(`/admin/sla-rules/${id}`, patch);
      return unwrap<SLARule>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-rule", id] });
      qc.invalidateQueries({ queryKey: ["sla-rules"] });
    },
  });
}

export function useDeleteSLARule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/sla-rules/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-rules"] });
    },
  });
}

export function useSetSLARuleActive() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, active }: { id: number; active: boolean }) => {
      await api.post(
        `/admin/sla-rules/${id}/${active ? "activate" : "deactivate"}`
      );
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["sla-rules"] });
    },
  });
}

export function useActivateSLARule() {
  const m = useSetSLARuleActive();
  return {
    ...m,
    mutate: (id: number) => m.mutate({ id, active: true }),
    mutateAsync: (id: number) => m.mutateAsync({ id, active: true }),
  };
}

export function useDeactivateSLARule() {
  const m = useSetSLARuleActive();
  return {
    ...m,
    mutate: (id: number) => m.mutate({ id, active: false }),
    mutateAsync: (id: number) => m.mutateAsync({ id, active: false }),
  };
}
