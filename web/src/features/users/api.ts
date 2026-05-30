import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { UserInfo } from "@/lib/types";

export interface UserFilters {
  page: number;
  page_size: number;
  search?: string;
  role?: string;
}

export interface UserPage {
  items: UserInfo[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export function useUsers(filters: UserFilters) {
  return useQuery({
    queryKey: ["users", filters],
    queryFn: async (): Promise<UserPage> => {
      const res = await api.get("/users", {
        params: {
          page: filters.page,
          page_size: filters.page_size,
          search: filters.search || undefined,
          role: filters.role || undefined,
        },
      });
      // The list endpoint returns { success, data: UserInfo[], meta: {...} }.
      const body = res.data as {
        data?: UserInfo[];
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

export function useUser(id: number | undefined) {
  return useQuery({
    queryKey: ["user", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/users/${id}`);
      return unwrap<UserInfo>(res.data);
    },
  });
}

export interface CreateUserInput {
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  password: string;
  role: string;
  is_active: boolean;
  customer_id?: number;
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateUserInput) => {
      const res = await api.post("/users", input);
      return unwrap<UserInfo>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export interface UpdateUserInput {
  email?: string;
  username?: string;
  first_name?: string;
  last_name?: string;
  role?: string;
  is_active?: boolean;
}

export function useUpdateUser(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (patch: UpdateUserInput) => {
      const res = await api.put(`/users/${id}`, patch);
      return unwrap<UserInfo>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["user", id] });
      qc.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/users/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export function useSetUserActive(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (active: boolean) => {
      await api.post(`/users/${id}/${active ? "activate" : "deactivate"}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["user", id] });
      qc.invalidateQueries({ queryKey: ["users"] });
    },
  });
}
