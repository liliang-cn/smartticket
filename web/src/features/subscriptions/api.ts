import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { Subscription } from "@/lib/types";

export interface SubscriptionFilters {
  page: number;
  page_size: number;
  customer_id?: number;
  status?: string;
}

export interface SubscriptionPage {
  items: Subscription[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface CreateSubscriptionInput {
  customer_id: number;
  product_id: number;
  sla_template_id?: number | null;
  plan?: string;
  billing_unit?: string;
  node_count?: number;
  billing_period?: string;
  starts_at?: string;
  expires_at?: string;
  status?: "active" | "expired" | "cancelled";
  unit_price?: number;
  currency?: string;
  notes?: string;
}

export function useSubscriptions(filters: SubscriptionFilters) {
  return useQuery({
    queryKey: ["subscriptions", filters],
    queryFn: async (): Promise<SubscriptionPage> => {
      const res = await api.get("/admin/subscriptions", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          customer_id: filters.customer_id || undefined,
          status: filters.status || undefined,
        },
      });
      // The list endpoint returns { success, data: Subscription[], meta: {...} }.
      const body = res.data as {
        data?: Subscription[];
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

export function useSubscription(id: number | undefined) {
  return useQuery({
    queryKey: ["subscription", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/admin/subscriptions/${id}`);
      return unwrap<Subscription>(res.data);
    },
  });
}

export function useCreateSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateSubscriptionInput) => {
      const res = await api.post("/admin/subscriptions", input);
      return unwrap<Subscription>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["subscriptions"] });
    },
  });
}

export function useUpdateSubscription(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<CreateSubscriptionInput>) => {
      const res = await api.put(`/admin/subscriptions/${id}`, patch);
      return unwrap<Subscription>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["subscription", id] });
      qc.invalidateQueries({ queryKey: ["subscriptions"] });
    },
  });
}

export function useDeleteSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/subscriptions/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["subscriptions"] });
    },
  });
}
