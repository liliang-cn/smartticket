import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

/** Deployment-wide AI feature toggles (admin-configurable). */
export interface AISettings {
  enabled: boolean;
  suggest_replies: boolean;
  knowledge_ai: boolean;
  auto_classify: boolean;
  reply_instructions: string;
  // Auto-resolve controls (new in competitive-parity)
  auto_reply_enabled: boolean;
  auto_reply_confidence: number; // 0..1
  auto_resolve_enabled: boolean;
  max_auto_replies_per_ticket: number; // >= 1
  auto_summarize_on_resolve: boolean;
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

/** The structured payload returned by the AI suggest-reply endpoint. */
export interface SuggestReplyResult {
  reply: string;
  confidence: number;
  needs_clarification: boolean;
  used_kb: boolean;
  sources: string[];
}

/** Ask the AI agent to draft a reply for a ticket (team only). */
export function useSuggestReply(ticketId: number) {
  return useMutation({
    mutationFn: async (): Promise<SuggestReplyResult> => {
      const res = await api.post(`/tickets/${ticketId}/suggest-reply`);
      const d = unwrap<SuggestReplyResult>(res.data);
      // Normalise: backend guarantees "reply" key; sources may be null.
      return {
        reply: d.reply ?? "",
        confidence: typeof d.confidence === "number" ? d.confidence : 0,
        needs_clarification: !!d.needs_clarification,
        used_kb: !!d.used_kb,
        sources: Array.isArray(d.sources) ? d.sources : [],
      };
    },
  });
}
