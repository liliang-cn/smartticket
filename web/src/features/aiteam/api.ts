import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────

export type AgentName = "Triage" | "Sentinel" | "Researcher" | "Reviewer" | "Drafter";
export type SuggestionStatus = "pending" | "done" | "adopted" | "dismissed" | "failed";

export interface AISuggestion {
  id: number;
  ticket_id: number;
  agent_name: AgentName;
  status: SuggestionStatus;
  /** Confidence 0–1, may be absent on pending/failed suggestions. */
  confidence: number;
  /** Raw JSON string — parse per-agent client-side. */
  payload: string;
  adopted_by?: number | null;
  resolved_at?: string | null;
  created_at: string;
}

// Per-agent parsed payload shapes

export interface TriagePayload {
  priority: string;
  severity: string;
  category: string;
  suggested_team_id?: number;
  reasoning: string;
  confidence: number;
}

export interface SentinelPayload {
  sentiment: string;
  churn_risk: string;
  sla_breach_risk: string;
  escalate: boolean;
  reasoning: string;
  confidence: number;
}

export interface KBCitation {
  title: string;
  snippet: string;
}

export interface SimilarTicket {
  id: number;
  title: string;
  resolution: string;
  merge_candidate: boolean;
  score: number;
}

export interface ResearcherPayload {
  kb_citations: KBCitation[];
  similar_tickets: SimilarTicket[];
  suggested_resolution: string;
  confidence: number;
}

export interface ReviewIssue {
  type: string;
  severity: string;
  note: string;
}

export interface ReviewerPayload {
  issues: ReviewIssue[];
  revised_draft?: string;
  approve: boolean;
  confidence: number;
}

export interface DrafterPayload {
  reply: string;
  confidence: number;
}

// ─── Query keys ───────────────────────────────────────────────────────────────

export function suggestionsKey(ticketId: number) {
  return ["ai-suggestions", ticketId] as const;
}

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useSuggestions(ticketId: number) {
  return useQuery({
    queryKey: suggestionsKey(ticketId),
    enabled: ticketId > 0,
    queryFn: async (): Promise<AISuggestion[]> => {
      const res = await api.get(`/tickets/${ticketId}/ai/suggestions`);
      const body = res.data as { suggestions?: AISuggestion[] };
      return body.suggestions ?? [];
    },
  });
}

export function useRunResearcher(ticketId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<AISuggestion> => {
      const res = await api.post(`/tickets/${ticketId}/ai/research`);
      return res.data as AISuggestion;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: suggestionsKey(ticketId) });
    },
  });
}

export function useRunReviewer(ticketId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (draft: string): Promise<AISuggestion> => {
      const res = await api.post(`/tickets/${ticketId}/ai/review`, { draft });
      return res.data as AISuggestion;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: suggestionsKey(ticketId) });
    },
  });
}

export function useRunDraft(ticketId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (): Promise<AISuggestion> => {
      const res = await api.post(`/tickets/${ticketId}/ai/draft`);
      return res.data as AISuggestion;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: suggestionsKey(ticketId) });
    },
  });
}

export function useAdoptSuggestion(ticketId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (sid: number) => {
      const res = await api.post(`/tickets/${ticketId}/ai/suggestions/${sid}/adopt`);
      return res.data as { adopted: boolean };
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: suggestionsKey(ticketId) });
    },
  });
}

export function useDismissSuggestion(ticketId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (sid: number) => {
      const res = await api.post(`/tickets/${ticketId}/ai/suggestions/${sid}/dismiss`);
      return res.data as { dismissed: boolean };
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: suggestionsKey(ticketId) });
    },
  });
}
