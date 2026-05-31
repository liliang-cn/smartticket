import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

export interface Service {
  id: number;
  product_id: number;
  name: string;
  code: string;
  description: string;
  type: string;
  status: string;
  availability: string;
  support_channels: string[];
  escalation_rules: Record<string, unknown>;
  configuration: Record<string, unknown>;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface ServiceFilters {
  page: number;
  page_size: number;
  search?: string;
  status?: string;
  category?: string;
  is_managed?: boolean;
}

export interface ServicePage {
  items: Service[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useServices(filters: ServiceFilters) {
  return useQuery({
    queryKey: ["services", filters],
    queryFn: async (): Promise<ServicePage> => {
      const res = await api.get("/admin/services", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
          status: filters.status || undefined,
          category: filters.category || undefined,
          is_managed: filters.is_managed,
        },
      });
      // The list endpoint returns { success, data: Service[], meta: {...} }.
      const body = res.data as {
        data?: Service[];
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

export function useService(id: number | undefined) {
  return useQuery({
    queryKey: ["service", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/admin/services/${id}`);
      return unwrap<Service>(res.data);
    },
  });
}

export interface CreateServiceInput {
  product_id: number;
  name: string;
  code?: string;
  description?: string;
  type?: string;
  status?: string;
  availability?: string;
  support_channels?: string[];
  tags?: string[];
}

export function useCreateService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateServiceInput) => {
      const res = await api.post("/admin/services", input);
      return unwrap<Service>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

export function useUpdateService(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<CreateServiceInput>) => {
      const res = await api.put(`/admin/services/${id}`, patch);
      return unwrap<Service>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["service", id] });
      qc.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

export function useDeleteService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/services/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

export function useActivateService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post(`/admin/services/${id}/activate`);
      return unwrap<Service>(res.data);
    },
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["service", id] });
      qc.invalidateQueries({ queryKey: ["services"] });
    },
  });
}

export function useDeactivateService() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post(`/admin/services/${id}/deactivate`);
      return unwrap<Service>(res.data);
    },
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["service", id] });
      qc.invalidateQueries({ queryKey: ["services"] });
    },
  });
}
