import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { Service } from "@/features/services/api";

export interface Product {
  id: number;
  name: string;
  code: string;
  description: string;
  category: string;
  version: string;
  status: string;
  is_managed: boolean;
  support_level: string;
  documentation: string;
  configuration: Record<string, unknown>;
  tags: string[];
  created_at: string;
  updated_at: string;
  services?: Service[];
}

export interface ProductFilters {
  page: number;
  page_size: number;
  search?: string;
  status?: string;
  category?: string;
  is_managed?: boolean;
}

export interface ProductPage {
  items: Product[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useProducts(filters: ProductFilters) {
  return useQuery({
    queryKey: ["products", filters],
    queryFn: async (): Promise<ProductPage> => {
      const res = await api.get("/admin/products", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
          status: filters.status || undefined,
          category: filters.category || undefined,
          is_managed: filters.is_managed,
        },
      });
      // The list endpoint returns { success, data: Product[], meta: {...} }.
      const body = res.data as {
        data?: Product[];
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

export function useProduct(id: number | undefined) {
  return useQuery({
    queryKey: ["product", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/admin/products/${id}`);
      return unwrap<Product>(res.data);
    },
  });
}

export interface CreateProductInput {
  name: string;
  code?: string;
  description?: string;
  category?: string;
  version?: string;
  status?: string;
  is_managed?: boolean;
  support_level?: string;
  documentation?: string;
  tags?: string[];
}

export function useCreateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateProductInput) => {
      const res = await api.post("/admin/products", input);
      return unwrap<Product>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["products"] });
    },
  });
}

export function useUpdateProduct(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: Partial<CreateProductInput>) => {
      const res = await api.put(`/admin/products/${id}`, patch);
      return unwrap<Product>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["product", id] });
      qc.invalidateQueries({ queryKey: ["products"] });
    },
  });
}

export function useDeleteProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/products/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["products"] });
    },
  });
}

export function useActivateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post(`/admin/products/${id}/activate`);
      return unwrap<Product>(res.data);
    },
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["product", id] });
      qc.invalidateQueries({ queryKey: ["products"] });
    },
  });
}

export function useDeactivateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post(`/admin/products/${id}/deactivate`);
      return unwrap<Product>(res.data);
    },
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["product", id] });
      qc.invalidateQueries({ queryKey: ["products"] });
    },
  });
}
