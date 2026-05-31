import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { api, unwrap } from "@/lib/api";

export type NotificationType =
  | "ticket_reply"
  | "ticket_assigned"
  | "ticket_status"
  | "ticket_created";

export interface Notification {
  id: number;
  user_id: number;
  type: NotificationType;
  title: string;
  body: string;
  ref_type: string;
  ref_id: number;
  is_read: boolean;
  created_at: number | string;
}

/** Poll the unread badge count every 30s. Returns 0 when unavailable. */
export function useUnreadCount() {
  return useQuery({
    queryKey: ["notifications-unread-count"],
    refetchInterval: 30000,
    queryFn: async () => {
      const res = await api.get("/notifications/unread-count");
      const data = unwrap<{ count: number }>(res.data);
      return data?.count ?? 0;
    },
  });
}

/**
 * Fetch the recent notification list. Only enabled when the dropdown is open
 * (pass `enabled`) to avoid background fetching the full list.
 */
export function useNotifications(unreadOnly = false, enabled = false) {
  return useQuery({
    queryKey: ["notifications", unreadOnly],
    enabled,
    queryFn: async () => {
      const res = await api.get("/notifications", {
        params: {
          page: 1,
          page_size: 15,
          ...(unreadOnly ? { unread: true } : {}),
        },
      });
      return unwrap<Notification[]>(res.data) ?? [];
    },
  });
}

function invalidate(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ["notifications"] });
  qc.invalidateQueries({ queryKey: ["notifications-unread-count"] });
}

export function useMarkRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.post(`/notifications/${id}/read`);
    },
    onSuccess: () => invalidate(qc),
  });
}

export function useMarkAllRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post("/notifications/read-all");
    },
    onSuccess: () => invalidate(qc),
  });
}
