import { useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

export interface AnalyticsBucket {
  name: string;
  count: number;
}

export interface AnalyticsEvent {
  event_type: string;
  path: string;
  referrer: string;
  source: string;
  target: string;
  device_type: string;
  created_at: number;
}

export interface AnalyticsSummary {
  days: number;
  total_events: number;
  pageviews: number;
  clicks: number;
  unique_visitors: number;
  top_referrers: AnalyticsBucket[];
  top_sources: AnalyticsBucket[];
  top_paths: AnalyticsBucket[];
  top_targets: AnalyticsBucket[];
  recent_events: AnalyticsEvent[];
}

export function useAnalyticsSummary(days = 30) {
  return useQuery({
    queryKey: ["analytics-summary", days],
    queryFn: async () => {
      const res = await api.get(`/analytics/summary?days=${days}`);
      return unwrap<AnalyticsSummary>(res.data);
    },
    staleTime: 60_000,
  });
}
