import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { Customer, CustomerUser } from "@/lib/types";

export interface CustomerFilters {
  page: number;
  page_size: number;
  search?: string;
  is_active?: boolean;
}

export interface CustomerPage {
  items: Customer[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useCustomers(filters: CustomerFilters) {
  return useQuery({
    queryKey: ["customers", filters],
    queryFn: async (): Promise<CustomerPage> => {
      const res = await api.get("/customers", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
          is_active: filters.is_active,
        },
      });
      // The list endpoint returns { success, data: Customer[], meta: {...} }.
      const body = res.data as {
        data?: Customer[];
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

export function useCustomer(id: number | undefined) {
  return useQuery({
    queryKey: ["customer", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/customers/${id}`);
      return unwrap<Customer>(res.data);
    },
  });
}

export function useCustomerUsers(id: number | undefined) {
  return useQuery({
    queryKey: ["customer-users", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/customers/${id}/users`);
      return unwrap<CustomerUser[]>(res.data) ?? [];
    },
  });
}

export interface CreateCustomerInput {
  name: string;
  code?: string;
  domain?: string;
  description?: string;
  is_active?: boolean;
}

export function useCreateCustomer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateCustomerInput) => {
      const res = await api.post("/customers", input);
      return unwrap<Customer>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}

export function useUpdateCustomer(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<CreateCustomerInput>) => {
      const res = await api.put(`/customers/${id}`, patch);
      return unwrap<Customer>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customer", id] });
      qc.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}

export function useDeleteCustomer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/customers/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["customers"] });
    },
  });
}
