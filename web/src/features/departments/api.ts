import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface Department {
  id: number;
  name: string;
  parent_id: number | null;
  manager_id: number | null;
  manager?: {
    id: number;
    first_name: string;
    last_name: string;
    username: string;
    email: string;
  } | null;
}

export interface CreateDepartmentInput {
  name: string;
  parent_id?: number | null;
  manager_id?: number | null;
}

export interface UpdateDepartmentInput {
  name?: string;
  parent_id?: number | null;
  manager_id?: number | null;
}

const QK = ["departments"] as const;

export function useDepartments() {
  return useQuery({
    queryKey: QK,
    queryFn: async () => {
      const res = await api.get("/admin/departments");
      const payload = res.data as { departments?: Department[] };
      return payload.departments ?? [];
    },
  });
}

export function useCreateDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateDepartmentInput): Promise<Department> => {
      const res = await api.post("/admin/departments", input);
      const payload = res.data as { department?: Department };
      return payload.department!;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}

export function useUpdateDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      ...input
    }: UpdateDepartmentInput & { id: number }) => {
      const res = await api.put(`/admin/departments/${id}`, input);
      return res.data as { updated: boolean };
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}

export function useDeleteDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/departments/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}
