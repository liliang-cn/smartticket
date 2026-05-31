import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";
import type { LLMProvider, ProviderInput, TestResult } from "./types";

// Providers are returned unpaginated as { success, data: [...] }.

export function useProviders() {
  return useQuery({
    queryKey: ["llm-providers"],
    queryFn: async () => {
      const res = await api.get("/llm/providers");
      return unwrap<LLMProvider[]>(res.data) ?? [];
    },
  });
}

export function useProvider(id: number | undefined) {
  return useQuery({
    queryKey: ["llm-providers", id],
    enabled: id != null,
    queryFn: async () => {
      const res = await api.get(`/llm/providers/${id}`);
      return unwrap<LLMProvider>(res.data);
    },
  });
}

export function useCreateProvider() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ProviderInput) => {
      const res = await api.post("/llm/providers", input);
      return unwrap<LLMProvider>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["llm-providers"] });
    },
  });
}

export function useUpdateProvider(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: ProviderInput) => {
      const res = await api.put(`/llm/providers/${id}`, input);
      return unwrap<LLMProvider>(res.data);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["llm-providers"] });
    },
  });
}

export function useDeleteProvider() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/llm/providers/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["llm-providers"] });
    },
  });
}

export function useTestProvider() {
  return useMutation({
    mutationFn: async (id: number) => {
      const res = await api.post(`/llm/providers/${id}/test`);
      return unwrap<TestResult>(res.data);
    },
  });
}
