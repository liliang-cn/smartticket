import { useMutation, useQuery } from "@tanstack/react-query";
import axios from "axios";
import { api, unwrap } from "@/lib/api";

/** Aggregate CSAT statistics returned by GET /api/v1/survey/stats */
export interface SurveyStats {
  sent_count: number;
  response_count: number;
  response_rate: number; // 0..1
  average_rating: number; // 0 when no responses
}

/** Public survey view returned by GET /api/v1/survey/:token */
export interface SurveyPublic {
  ticket_id: number;
  rating: number;
  responded: boolean;
}

/** Fetch CSAT aggregate stats (team/admin, authenticated). */
export function useSurveyStats() {
  return useQuery({
    queryKey: ["survey-stats"],
    queryFn: async () => {
      const res = await api.get("/survey/stats");
      return unwrap<SurveyStats>(res.data);
    },
    staleTime: 60_000,
  });
}

/** Fetch a public survey by token (no auth). */
export function useSurveyPublic(token: string) {
  return useQuery({
    queryKey: ["survey-public", token],
    queryFn: async () => {
      // Use a plain axios instance — no JWT, no interceptors.
      const res = await axios.get(`/api/v1/survey/${token}`);
      return unwrap<SurveyPublic>(res.data);
    },
    retry: false,
  });
}

/** Submit a rating + comment for a public survey token (no auth). */
export function useSubmitSurvey(token: string) {
  return useMutation({
    mutationFn: async (body: { rating: number; comment: string }) => {
      await axios.post(`/api/v1/survey/${token}`, body);
    },
  });
}
