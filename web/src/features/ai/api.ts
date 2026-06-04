import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

/** Deployment-wide AI feature toggles (admin-configurable). */
export interface AISettings {
  enabled: boolean;
  suggest_replies: boolean;
  knowledge_ai: boolean;
  auto_classify: boolean;
  reply_instructions: string;
}

/** Read the AI settings (any authenticated user — drives which AI affordances show). */
export function useAISettings() {
  return useQuery({
    queryKey: ["ai-settings"],
    queryFn: async () => {
      const res = await api.get("/settings/ai");
      return unwrap<AISettings>(res.data);
    },
    staleTime: 60_000,
  });
}

/** Patch the AI settings (admin only). */
export function useUpdateAISettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: Partial<AISettings>) => {
      const res = await api.put("/settings/ai", body);
      return unwrap<AISettings>(res.data);
    },
    onSuccess: (data) => {
      qc.setQueryData(["ai-settings"], data);
      qc.invalidateQueries({ queryKey: ["ai-settings"] });
    },
  });
}

/** Ask the AI agent to draft a reply for a ticket (team only). */
export function useSuggestReply(ticketId: number) {
  return useMutation({
    mutationFn: async (): Promise<string> => {
      const res = await api.post(`/tickets/${ticketId}/suggest-reply`);
      return unwrap<{ suggestion: string }>(res.data).suggestion;
    },
  });
}
