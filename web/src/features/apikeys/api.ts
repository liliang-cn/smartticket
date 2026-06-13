import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface ApiKey {
  id: number;
  name: string;
  key_prefix: string;
  user_id: number;
  is_active: boolean;
  expires_at: number | null;
  last_used_at: number | null;
  created_at: number;
}

export interface CreateApiKeyInput {
  name: string;
  user_id: number;
  expires_at?: number;
}

export interface CreateApiKeyResult {
  key: string;
  api_key: ApiKey;
}

const QK = ["api-keys"] as const;

export function useApiKeys() {
  return useQuery({
    queryKey: QK,
    queryFn: async () => {
      const res = await api.get("/admin/api-keys");
      const payload = res.data as { api_keys?: ApiKey[] };
      return payload.api_keys ?? [];
    },
  });
}

export function useCreateApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateApiKeyInput): Promise<CreateApiKeyResult> => {
      const res = await api.post("/admin/api-keys", input);
      return res.data as CreateApiKeyResult;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}

export function useRevokeApiKey() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/admin/api-keys/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: QK });
    },
  });
}
