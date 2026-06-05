import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

export interface Macro {
  id: number;
  title: string;
  category: string;
  body: string;
  actions: string;
  shared: boolean;
  owner_id: number;
  usage_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateMacroInput {
  title: string;
  category: string;
  body: string;
  actions?: string;
  shared: boolean;
}

export interface UpdateMacroInput {
  title?: string;
  category?: string;
  body?: string;
  actions?: string;
  shared?: boolean;
}

export function useMacros() {
  return useQuery({
    queryKey: ["macros"],
    queryFn: async () => {
      const res = await api.get("/macros");
      return unwrap<Macro[]>(res.data) ?? [];
    },
  });
}

export function useCreateMacro() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateMacroInput) => {
      const res = await api.post("/macros", input);
      return unwrap<Macro>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["macros"] });
    },
  });
}

export function useUpdateMacro(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateMacroInput) => {
      const res = await api.put(`/macros/${id}`, input);
      return unwrap<Macro>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["macros"] });
    },
  });
}

export function useDeleteMacro() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/macros/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["macros"] });
    },
  });
}
